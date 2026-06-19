package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"oil/config"
	"oil/infras/asynq"
	"oil/infras/kafka"
	"oil/infras/otel"
	"oil/infras/postgres"
	bookingModel "oil/internal/domains/booking/model"
	"oil/internal/domains/booking/model/dto"
	bookingRepo "oil/internal/domains/booking/repository"
	paymentSvc "oil/internal/domains/payment/service"
	scheduleModel "oil/internal/domains/schedule/model"
	scheduleRepo "oil/internal/domains/schedule/repository"
	seatModel "oil/internal/domains/seat/model"
	seatRepo "oil/internal/domains/seat/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	gModel "oil/shared/model"
	"oil/shared/seatlock"
	"oil/shared/timezone"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

const (
	defaultHoldTTLSeconds = 600
	defaultExpireMinutes  = 10
)

type Booking interface {
	Create(ctx context.Context, userID string, req dto.CreateBookingRequest) (dto.BookingResponse, error)
	GetByID(ctx context.Context, id, userID string, isAdmin bool) (dto.BookingResponse, error)
	GetUserBookings(ctx context.Context, userID string, params gDto.QueryParams) (dto.GetBookingsResponse, error)
	SeatMap(ctx context.Context, scheduleID string) (dto.SeatMapResponse, error)
	Confirm(ctx context.Context, bookingID string) error
	Release(ctx context.Context, bookingID string) error
}

type serviceImpl struct {
	bookingRepo     bookingRepo.Booking
	bookingSeatRepo bookingRepo.Seat
	scheduleRepo    scheduleRepo.Schedule
	seatRepo        seatRepo.Seat
	locker          seatlock.Locker
	paymentSvc      paymentSvc.Payment
	asynqClient     asynq.Client
	kafkaClient     kafka.Client
	db              *postgres.Connection
	cfg             *config.Config
	otel            otel.Otel
}

func New(
	bookingRepo bookingRepo.Booking,
	bookingSeatRepo bookingRepo.Seat,
	scheduleRepo scheduleRepo.Schedule,
	seatRepo seatRepo.Seat,
	locker seatlock.Locker,
	paymentSvc paymentSvc.Payment,
	asynqClient asynq.Client,
	kafkaClient kafka.Client,
	db *postgres.Connection,
	cfg *config.Config,
	otel otel.Otel,
) Booking {
	return &serviceImpl{
		bookingRepo:     bookingRepo,
		bookingSeatRepo: bookingSeatRepo,
		scheduleRepo:    scheduleRepo,
		seatRepo:        seatRepo,
		locker:          locker,
		paymentSvc:      paymentSvc,
		asynqClient:     asynqClient,
		kafkaClient:     kafkaClient,
		db:              db,
		cfg:             cfg,
		otel:            otel,
	}
}

func (s *serviceImpl) holdTTL() int {
	if s.cfg.Payment.ExpireMinutes > 0 {
		return s.cfg.Payment.ExpireMinutes * 60
	}

	return defaultHoldTTLSeconds
}

func (s *serviceImpl) expireMinutes() int {
	if s.cfg.Payment.ExpireMinutes > 0 {
		return s.cfg.Payment.ExpireMinutes
	}

	return defaultExpireMinutes
}

func (s *serviceImpl) Create(ctx context.Context, userID string, req dto.CreateBookingRequest) (res dto.BookingResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	seatIDs := dedupe(req.SeatIDs)

	schedule, err := s.scheduleRepo.Get(ctx, shared.FilterByID(req.ScheduleID, scheduleModel.FieldID, scheduleModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule.ID == "" {
		return res, failure.NotFound("schedule not found")
	}

	if schedule.Status != scheduleModel.StatusScheduled {
		return res, failure.Conflict("schedule is not open for booking")
	}

	if schedule.StartTime.Before(timezone.Now()) {
		return res, failure.BadRequestFromString("schedule has already started")
	}

	if err = s.validateSeats(ctx, schedule.StudioID, req.ScheduleID, seatIDs); err != nil {
		return res, err
	}

	// Lapis 1: atomic Redis hold (fast, all-or-nothing).
	ok, err := s.locker.Hold(ctx, req.ScheduleID, userID, seatIDs, s.holdTTL())
	if err != nil {
		return res, fmt.Errorf("failed to hold seats: %w", err)
	}

	if !ok {
		return res, failure.Conflict("one or more seats are no longer available")
	}

	booking, seats := s.buildBooking(userID, schedule, seatIDs)

	if err = s.persistBooking(ctx, booking, seats); err != nil {
		// Roll back the Redis hold so the seats free up immediately.
		_ = s.locker.Release(ctx, req.ScheduleID, seatIDs)

		return res, err
	}

	payment, err := s.paymentSvc.CreateCharge(ctx, booking.ID, booking.TotalAmount, booking.ID)
	if err != nil {
		log.Error().Err(err).Str("booking_id", booking.ID).Msg("failed to create charge")

		return res, fmt.Errorf("failed to create charge: %w", err)
	}

	s.enqueueRelease(ctx, booking.ID, *booking.ExpiresAt)

	res.FromModel(booking)
	res.SetSeats(seats)
	res.PaymentReference = payment.Reference

	return res, nil
}

func (s *serviceImpl) validateSeats(ctx context.Context, studioID, scheduleID string, seatIDs []string) error {
	// All seats must exist and belong to this schedule's studio.
	seats, err := s.seatRepo.GetAll(ctx, gDto.QueryParams{}, gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: seatModel.FieldStudioID, Table: seatModel.TableName, Value: studioID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: seatModel.FieldID, Table: seatModel.TableName, Value: seatIDs, Operator: gDto.FilterOperatorIn},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to validate seats: %w", err)
	}

	if len(seats) != len(seatIDs) {
		return failure.BadRequestFromString("one or more seats are invalid for this schedule")
	}

	// None of the seats may already be booked for this schedule.
	booked, err := s.bookingSeatRepo.GetAll(ctx, gDto.QueryParams{}, gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: bookingModel.SeatFieldScheduleID, Table: bookingModel.SeatTableName, Value: scheduleID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: bookingModel.SeatFieldSeatID, Table: bookingModel.SeatTableName, Value: seatIDs, Operator: gDto.FilterOperatorIn},
			gDto.Filter{Field: bookingModel.SeatFieldStatus, ArgName: "booked_status", Table: bookingModel.SeatTableName, Value: bookingModel.SeatStatusBooked, Operator: gDto.FilterOperatorEq},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to check booked seats: %w", err)
	}

	if len(booked) > 0 {
		return failure.Conflict("one or more seats are already booked")
	}

	return nil
}

func (s *serviceImpl) buildBooking(userID string, schedule scheduleModel.Schedule, seatIDs []string) (bookingModel.Booking, []bookingModel.BookingSeat) {
	now := timezone.Now()
	expiresAt := now.Add(time.Duration(s.expireMinutes()) * time.Minute)
	bookingID := uuid.NewString()

	meta := gModel.Metadata{CreatedAt: now, ModifiedAt: now, CreatedBy: userID, ModifiedBy: userID}

	booking := bookingModel.Booking{
		ID:          bookingID,
		BookingCode: generateBookingCode(),
		UserID:      userID,
		ScheduleID:  schedule.ID,
		Status:      bookingModel.StatusPending,
		TotalAmount: schedule.Price * float64(len(seatIDs)),
		SeatCount:   len(seatIDs),
		ExpiresAt:   &expiresAt,
		Active:      true,
		Metadata:    meta,
	}

	seats := make([]bookingModel.BookingSeat, len(seatIDs))
	for i, seatID := range seatIDs {
		seats[i] = bookingModel.BookingSeat{
			ID:         uuid.NewString(),
			BookingID:  bookingID,
			ScheduleID: schedule.ID,
			SeatID:     seatID,
			Status:     bookingModel.SeatStatusHeld,
			Price:      schedule.Price,
			Metadata:   meta,
		}
	}

	return booking, seats
}

func (s *serviceImpl) persistBooking(ctx context.Context, booking bookingModel.Booking, seats []bookingModel.BookingSeat) (err error) {
	tx, err := s.db.Write.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.bookingRepo.InsertTx(ctx, tx, booking); err != nil {
		return fmt.Errorf("failed to insert booking: %w", err)
	}

	if err = s.bookingSeatRepo.InsertBulkTx(ctx, tx, seats); err != nil {
		return fmt.Errorf("failed to insert booking seats: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit booking: %w", err)
	}

	return nil
}

// Confirm flips the booking's held seats to booked inside a transaction. The
// booking_seats partial unique index guards against double-booking: if any seat
// was booked elsewhere, the UPDATE raises a unique violation and we abort.
func (s *serviceImpl) Confirm(ctx context.Context, bookingID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.Confirm")
	defer scope.End()
	defer scope.TraceIfError(err)

	booking, err := s.bookingRepo.Get(ctx, shared.FilterByID(bookingID, bookingModel.FieldID, bookingModel.TableName))
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if booking.ID == "" {
		return failure.NotFound("booking not found")
	}

	if booking.Status != bookingModel.StatusPending {
		// Idempotent: already confirmed/expired.
		return nil
	}

	seatIDs, err := s.heldSeatIDs(ctx, bookingID)
	if err != nil {
		return err
	}

	if err = s.confirmTx(ctx, bookingID); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == constant.PqErrorCodeUniqueViolation {
			// A seat was taken despite the hold (e.g. Redis was flushed). Expire
			// this booking and free its seats; the DB stayed correct.
			log.Warn().Str("booking_id", bookingID).Msg("confirm conflicted on unique index; expiring booking")
			_ = s.Release(ctx, bookingID)

			return failure.Conflict("one or more seats were already booked")
		}

		return err
	}

	_ = s.locker.Release(ctx, booking.ScheduleID, seatIDs)
	s.publish(ctx, constant.TopicTicketSold, bookingID, map[string]any{
		"booking_id":  bookingID,
		"schedule_id": booking.ScheduleID,
		"user_id":     booking.UserID,
		"seat_ids":    seatIDs,
	})
	s.enqueue(ctx, constant.TaskSendETicket, map[string]string{"booking_id": bookingID}, timezone.Now())

	return nil
}

func (s *serviceImpl) confirmTx(ctx context.Context, bookingID string) (err error) {
	tx, err := s.db.Write.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := timezone.Now()

	seatUpdate := map[string]any{
		bookingModel.SeatFieldStatus: bookingModel.SeatStatusBooked,
		constant.FieldModifiedAt:     now,
		constant.FieldModifiedBy:     systemActor,
	}

	if err = s.bookingSeatRepo.UpdateTx(ctx, tx, seatUpdate, seatStatusFilter(bookingID, bookingModel.SeatStatusHeld)); err != nil {
		return fmt.Errorf("failed to book seats: %w", err)
	}

	bookingUpdate := map[string]any{
		bookingModel.FieldStatus: bookingModel.StatusConfirmed,
		constant.FieldModifiedAt: now,
		constant.FieldModifiedBy: systemActor,
	}

	if err = s.bookingRepo.UpdateTx(ctx, tx, bookingUpdate, shared.FilterByID(bookingID, bookingModel.FieldID, bookingModel.TableName)); err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit confirm: %w", err)
	}

	return nil
}

// Release expires a still-pending booking, frees its held seats, and restocks.
// Idempotent: a no-op if the booking is already confirmed or expired.
func (s *serviceImpl) Release(ctx context.Context, bookingID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.Release")
	defer scope.End()
	defer scope.TraceIfError(err)

	booking, err := s.bookingRepo.Get(ctx, shared.FilterByID(bookingID, bookingModel.FieldID, bookingModel.TableName))
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if booking.ID == "" || booking.Status != bookingModel.StatusPending {
		return nil
	}

	seatIDs, err := s.heldSeatIDs(ctx, bookingID)
	if err != nil {
		return err
	}

	now := timezone.Now()

	seatUpdate := map[string]any{
		bookingModel.SeatFieldStatus: bookingModel.SeatStatusReleased,
		constant.FieldModifiedAt:     now,
		constant.FieldModifiedBy:     systemActor,
	}

	if err = s.bookingSeatRepo.Update(ctx, seatUpdate, seatStatusFilter(bookingID, bookingModel.SeatStatusHeld)); err != nil {
		return fmt.Errorf("failed to release seats: %w", err)
	}

	bookingUpdate := map[string]any{
		bookingModel.FieldStatus: bookingModel.StatusExpired,
		constant.FieldModifiedAt: now,
		constant.FieldModifiedBy: systemActor,
	}

	if err = s.bookingRepo.Update(ctx, bookingUpdate, shared.FilterByID(bookingID, bookingModel.FieldID, bookingModel.TableName)); err != nil {
		return fmt.Errorf("failed to expire booking: %w", err)
	}

	_ = s.locker.Release(ctx, booking.ScheduleID, seatIDs)
	s.publish(ctx, constant.TopicSeatRestocked, booking.ScheduleID, map[string]any{
		"schedule_id": booking.ScheduleID,
		"seat_ids":    seatIDs,
		"reason":      "hold_expired",
	})

	return nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id, userID string, isAdmin bool) (res dto.BookingResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.GetByID")
	defer scope.End()
	defer scope.TraceIfError(err)

	booking, err := s.bookingRepo.Get(ctx, shared.FilterByID(id, bookingModel.FieldID, bookingModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get booking: %w", err)
	}

	if booking.ID == "" || (!isAdmin && booking.UserID != userID) {
		return res, failure.NotFound("booking not found")
	}

	seats, err := s.bookingSeatRepo.GetAll(ctx, gDto.QueryParams{}, shared.FilterByID(id, bookingModel.SeatFieldBookingID, bookingModel.SeatTableName))
	if err != nil {
		return res, fmt.Errorf("failed to get booking seats: %w", err)
	}

	res.FromModel(booking)
	res.SetSeats(seats)

	return res, nil
}

func (s *serviceImpl) GetUserBookings(ctx context.Context, userID string, params gDto.QueryParams) (res dto.GetBookingsResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.GetUserBookings")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(userID, bookingModel.FieldUserID, bookingModel.TableName)

	total, err := s.bookingRepo.Count(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to count bookings: %w", err)
	}

	bookings, err := s.bookingRepo.GetAll(ctx, params, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get bookings: %w", err)
	}

	res.FromModels(bookings, total, params.Limit)

	return res, nil
}

func (s *serviceImpl) SeatMap(ctx context.Context, scheduleID string) (res dto.SeatMapResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".booking.SeatMap")
	defer scope.End()
	defer scope.TraceIfError(err)

	schedule, err := s.scheduleRepo.Get(ctx, shared.FilterByID(scheduleID, scheduleModel.FieldID, scheduleModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule.ID == "" {
		return res, failure.NotFound("schedule not found")
	}

	seats, err := s.seatRepo.GetAll(ctx, gDto.QueryParams{}, shared.FilterByID(schedule.StudioID, seatModel.FieldStudioID, seatModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get seats: %w", err)
	}

	booked, err := s.bookingSeatRepo.GetAll(ctx, gDto.QueryParams{}, gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: bookingModel.SeatFieldScheduleID, Table: bookingModel.SeatTableName, Value: scheduleID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: bookingModel.SeatFieldStatus, ArgName: "booked_status", Table: bookingModel.SeatTableName, Value: bookingModel.SeatStatusBooked, Operator: gDto.FilterOperatorEq},
		},
	})
	if err != nil {
		return res, fmt.Errorf("failed to get booked seats: %w", err)
	}

	bookedSet := make(map[string]bool, len(booked))
	for _, b := range booked {
		bookedSet[b.SeatID] = true
	}

	seatIDs := make([]string, len(seats))
	for i, st := range seats {
		seatIDs[i] = st.ID
	}

	heldSet, err := s.locker.HeldSeatIDs(ctx, scheduleID, seatIDs)
	if err != nil {
		return res, fmt.Errorf("failed to read held seats: %w", err)
	}

	res = dto.SeatMapResponse{ScheduleID: scheduleID, TotalSeats: len(seats), Seats: make([]dto.SeatMapSeat, len(seats))}

	for i, st := range seats {
		status := "available"

		switch {
		case bookedSet[st.ID]:
			status = "booked"
			res.Booked++
		case heldSet[st.ID]:
			status = "held"
			res.Held++
		default:
			res.Available++
		}

		row := ""
		if st.RowLabel != nil {
			row = *st.RowLabel
		}

		res.Seats[i] = dto.SeatMapSeat{
			SeatID:    st.ID,
			SeatLabel: st.SeatLabel,
			RowLabel:  row,
			SeatType:  st.SeatType,
			Status:    status,
		}
	}

	return res, nil
}

func (s *serviceImpl) heldSeatIDs(ctx context.Context, bookingID string) ([]string, error) {
	seats, err := s.bookingSeatRepo.GetAll(ctx, gDto.QueryParams{}, seatStatusFilter(bookingID, bookingModel.SeatStatusHeld))
	if err != nil {
		return nil, fmt.Errorf("failed to get held seats: %w", err)
	}

	ids := make([]string, len(seats))
	for i, st := range seats {
		ids[i] = st.SeatID
	}

	return ids, nil
}

func (s *serviceImpl) enqueueRelease(ctx context.Context, bookingID string, processAt time.Time) {
	s.enqueue(ctx, constant.TaskReleaseHold, map[string]string{"booking_id": bookingID}, processAt)
}

func (s *serviceImpl) enqueue(ctx context.Context, taskType string, payload any, processAt time.Time) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("task", taskType).Msg("failed to marshal task payload")

		return
	}

	if _, err := s.asynqClient.Enqueue(ctx, taskType, body, processAt); err != nil {
		log.Error().Err(err).Str("task", taskType).Msg("failed to enqueue task")
	}
}

func (s *serviceImpl) publish(ctx context.Context, topic, key string, value any) {
	if !s.cfg.Kafka.Enable {
		return
	}

	if err := s.kafkaClient.SendMessages(ctx, topic, kafka.Message{Key: key, Value: value}); err != nil {
		log.Error().Err(err).Str("topic", topic).Msg("failed to publish event")
	}
}

const systemActor = "system"

// seatStatusFilter matches a booking's seats in a given status. The status
// filter uses a distinct ArgName so it never collides with a SET status=:status
// in the same UPDATE.
func seatStatusFilter(bookingID, status string) gDto.FilterGroup {
	return gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: bookingModel.SeatFieldBookingID, Value: bookingID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: bookingModel.SeatFieldStatus, ArgName: "current_status", Value: status, Operator: gDto.FilterOperatorEq},
		},
	}
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))

	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}

	return out
}

func generateBookingCode() string {
	return "BK-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:10])
}
