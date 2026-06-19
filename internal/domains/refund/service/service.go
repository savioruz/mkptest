package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"encoding/json"
	"fmt"

	"oil/config"
	"oil/infras/asynq"
	"oil/infras/kafka"
	"oil/infras/otel"
	gateway "oil/infras/payment"
	"oil/infras/postgres"
	bookingModel "oil/internal/domains/booking/model"
	bookingRepo "oil/internal/domains/booking/repository"
	paymentModel "oil/internal/domains/payment/model"
	paymentRepo "oil/internal/domains/payment/repository"
	"oil/internal/domains/refund/model"
	refundRepo "oil/internal/domains/refund/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const systemActor = "system"

type Refund interface {
	// RequestForBooking creates (idempotently) a customer-initiated PENDING refund and publishes refund.requested.
	RequestForBooking(ctx context.Context, bookingID, requestorID, reason string) (model.Refund, error)
	// CreateForCancelledSchedule fans out a refund request per confirmed booking on a cancelled schedule.
	CreateForCancelledSchedule(ctx context.Context, scheduleID string) error
	// Process settles a PENDING refund via the gateway, marks the booking refunded and releases its seats.
	Process(ctx context.Context, refundID string) error
}

type serviceImpl struct {
	refundRepo      refundRepo.Refund
	bookingRepo     bookingRepo.Booking
	bookingSeatRepo bookingRepo.Seat
	paymentRepo     paymentRepo.Payment
	gateway         gateway.Gateway
	kafkaClient     kafka.Client
	asynqClient     asynq.Client
	db              *postgres.Connection
	cfg             *config.Config
	otel            otel.Otel
}

func New(
	refundRepo refundRepo.Refund,
	bookingRepo bookingRepo.Booking,
	bookingSeatRepo bookingRepo.Seat,
	paymentRepo paymentRepo.Payment,
	gateway gateway.Gateway,
	kafkaClient kafka.Client,
	asynqClient asynq.Client,
	db *postgres.Connection,
	cfg *config.Config,
	otel otel.Otel,
) Refund {
	return &serviceImpl{
		refundRepo:      refundRepo,
		bookingRepo:     bookingRepo,
		bookingSeatRepo: bookingSeatRepo,
		paymentRepo:     paymentRepo,
		gateway:         gateway,
		kafkaClient:     kafkaClient,
		asynqClient:     asynqClient,
		db:              db,
		cfg:             cfg,
		otel:            otel,
	}
}

func (s *serviceImpl) refundTx(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := s.db.Write.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

func (s *serviceImpl) RequestForBooking(ctx context.Context, bookingID, requestorID, reason string) (res model.Refund, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".refund.RequestForBooking")
	defer scope.End()
	defer scope.TraceIfError(err)

	booking, err := s.bookingRepo.Get(ctx, shared.FilterByID(bookingID, bookingModel.FieldID, bookingModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get booking: %w", err)
	}

	// Hide other users' bookings behind a 404.
	if booking.ID == "" || booking.UserID != requestorID {
		return res, failure.NotFound("booking not found")
	}

	if booking.Status != bookingModel.StatusConfirmed {
		return res, failure.Conflict("only confirmed bookings can be refunded")
	}

	return s.createRefund(ctx, booking, model.InitiatedByCustomer, reason)
}

func (s *serviceImpl) CreateForCancelledSchedule(ctx context.Context, scheduleID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".refund.CreateForCancelledSchedule")
	defer scope.End()
	defer scope.TraceIfError(err)

	bookings, err := s.bookingRepo.GetAll(ctx, gDto.QueryParams{}, gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: bookingModel.FieldScheduleID, Table: bookingModel.TableName, Value: scheduleID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: bookingModel.FieldStatus, ArgName: "confirmed_status", Table: bookingModel.TableName, Value: bookingModel.StatusConfirmed, Operator: gDto.FilterOperatorEq},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list confirmed bookings: %w", err)
	}

	log.Info().Str("schedule_id", scheduleID).Int("bookings", len(bookings)).Msg("creating mass refunds for cancelled schedule")

	for i := range bookings {
		if _, err := s.createRefund(ctx, bookings[i], model.InitiatedByCinema, "schedule cancelled by cinema"); err != nil {
			// One failure must not block the rest; each refund is independent.
			log.Error().Err(err).Str("booking_id", bookings[i].ID).Msg("failed to create refund")
		}
	}

	return nil
}

// createRefund inserts a PENDING refund (idempotent by booking) and publishes refund.requested.
func (s *serviceImpl) createRefund(ctx context.Context, booking bookingModel.Booking, initiatedBy, reason string) (model.Refund, error) {
	// The DB CHECK was removed; the allowed initiators are validated here.
	if !constant.RefundInitiator(initiatedBy).Valid() {
		return model.Refund{}, failure.BadRequestFromString("invalid refund initiator")
	}

	idempotencyKey := "RF-" + booking.ID

	existing, err := s.refundRepo.Get(ctx, shared.FilterByID(idempotencyKey, model.FieldIdempotencyKey, model.TableName))
	if err != nil {
		return existing, fmt.Errorf("failed to check existing refund: %w", err)
	}

	if existing.ID != "" {
		return existing, nil
	}

	var paymentID *string

	payment, err := s.paymentRepo.Get(ctx, gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: paymentModel.FieldBookingID, Table: paymentModel.TableName, Value: booking.ID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: paymentModel.FieldStatus, ArgName: "success_status", Table: paymentModel.TableName, Value: paymentModel.StatusSuccess, Operator: gDto.FilterOperatorEq},
		},
	})
	if err != nil {
		return existing, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.ID != "" {
		paymentID = &payment.ID
	}

	now := timezone.Now()
	reasonCopy := reason
	refund := model.Refund{
		ID:             uuid.NewString(),
		BookingID:      booking.ID,
		PaymentID:      paymentID,
		Amount:         booking.TotalAmount,
		Status:         model.StatusPending,
		InitiatedBy:    initiatedBy,
		IdempotencyKey: idempotencyKey,
		Reason:         &reasonCopy,
		Metadata:       gModel.Metadata{CreatedAt: now, ModifiedAt: now, CreatedBy: systemActor, ModifiedBy: systemActor},
	}

	if err := s.refundRepo.Insert(ctx, refund); err != nil {
		return refund, fmt.Errorf("failed to insert refund: %w", err)
	}

	s.publish(ctx, constant.TopicRefundRequested, refund.ID, map[string]any{
		"refund_id":  refund.ID,
		"booking_id": booking.ID,
	})

	return refund, nil
}

func (s *serviceImpl) Process(ctx context.Context, refundID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".refund.Process")
	defer scope.End()
	defer scope.TraceIfError(err)

	refund, err := s.refundRepo.Get(ctx, shared.FilterByID(refundID, model.FieldID, model.TableName))
	if err != nil {
		return fmt.Errorf("failed to get refund: %w", err)
	}

	if refund.ID == "" || refund.Status != model.StatusPending {
		// Idempotent: already processed or unknown.
		return nil
	}

	reference := ""
	if refund.PaymentID != nil {
		payment, pErr := s.paymentRepo.Get(ctx, shared.FilterByID(*refund.PaymentID, paymentModel.FieldID, paymentModel.TableName))
		if pErr != nil {
			return fmt.Errorf("failed to get payment: %w", pErr)
		}

		reference = payment.Reference
	}

	if err = s.gateway.Refund(ctx, reference, refund.Amount); err != nil {
		log.Error().Err(err).Str("refund_id", refundID).Msg("gateway refund failed")
		_ = s.refundRepo.Update(ctx, map[string]any{
			model.FieldStatus:        model.StatusFailed,
			constant.FieldModifiedAt: timezone.Now(),
			constant.FieldModifiedBy: systemActor,
		}, shared.FilterByID(refundID, model.FieldID, model.TableName))

		return fmt.Errorf("gateway refund failed: %w", err)
	}

	if err = s.settle(ctx, refund); err != nil {
		return err
	}

	s.publish(ctx, constant.TopicSeatRestocked, refund.BookingID, map[string]any{
		"booking_id": refund.BookingID,
		"reason":     "refund",
	})
	s.enqueueNotify(ctx, refund.ID, refund.BookingID)

	return nil
}

// settle marks the refund completed, the booking refunded, and releases its
// booked seats — all in one transaction.
func (s *serviceImpl) settle(ctx context.Context, refund model.Refund) (err error) {
	tx, err := s.refundTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := timezone.Now()

	refundUpdate := map[string]any{
		model.FieldStatus:        model.StatusCompleted,
		model.FieldRefundedAt:    now,
		constant.FieldModifiedAt: now,
		constant.FieldModifiedBy: systemActor,
	}
	if err = s.refundRepo.UpdateTx(ctx, tx, refundUpdate, shared.FilterByID(refund.ID, model.FieldID, model.TableName)); err != nil {
		return fmt.Errorf("failed to complete refund: %w", err)
	}

	bookingUpdate := map[string]any{
		bookingModel.FieldStatus: bookingModel.StatusRefunded,
		constant.FieldModifiedAt: now,
		constant.FieldModifiedBy: systemActor,
	}
	if err = s.bookingRepo.UpdateTx(ctx, tx, bookingUpdate, shared.FilterByID(refund.BookingID, bookingModel.FieldID, bookingModel.TableName)); err != nil {
		return fmt.Errorf("failed to mark booking refunded: %w", err)
	}

	// Release booked seats so they return to the pool (frees the partial unique index).
	seatUpdate := map[string]any{
		bookingModel.SeatFieldStatus: bookingModel.SeatStatusReleased,
		constant.FieldModifiedAt:     now,
		constant.FieldModifiedBy:     systemActor,
	}
	seatFilter := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{Field: bookingModel.SeatFieldBookingID, Value: refund.BookingID, Operator: gDto.FilterOperatorEq},
			gDto.Filter{Field: bookingModel.SeatFieldStatus, ArgName: "current_status", Value: bookingModel.SeatStatusBooked, Operator: gDto.FilterOperatorEq},
		},
	}
	if err = s.bookingSeatRepo.UpdateTx(ctx, tx, seatUpdate, seatFilter); err != nil {
		return fmt.Errorf("failed to release booked seats: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit refund settlement: %w", err)
	}

	return nil
}

func (s *serviceImpl) publish(ctx context.Context, topic, key string, value any) {
	if !s.cfg.Kafka.Enable {
		return
	}

	if err := s.kafkaClient.SendMessages(ctx, topic, kafka.Message{Key: key, Value: value}); err != nil {
		log.Error().Err(err).Str("topic", topic).Msg("failed to publish event")
	}
}

func (s *serviceImpl) enqueueNotify(ctx context.Context, refundID, bookingID string) {
	body, _ := json.Marshal(map[string]string{"refund_id": refundID, "booking_id": bookingID})
	if _, err := s.asynqClient.Enqueue(ctx, constant.TaskNotifyRefund, body, timezone.Now()); err != nil {
		log.Error().Err(err).Msg("failed to enqueue refund notification")
	}
}
