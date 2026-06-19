package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"oil/config"
	"oil/infras/kafka"
	"oil/infras/otel"
	movieModel "oil/internal/domains/movie/model"
	movieRepo "oil/internal/domains/movie/repository"
	"oil/internal/domains/schedule/model"
	"oil/internal/domains/schedule/model/dto"
	scheduleRepo "oil/internal/domains/schedule/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	"oil/shared/timezone"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Schedule interface {
	Create(ctx context.Context, req dto.CreateScheduleRequest, userID string) (dto.ScheduleResponse, error)
	GetByID(ctx context.Context, id string) (dto.ScheduleResponse, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (dto.GetSchedulesResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateScheduleRequest, userID string) (dto.ScheduleResponse, error)
	Cancel(ctx context.Context, id, userID string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	scheduleRepo scheduleRepo.Schedule
	movieRepo    movieRepo.Movie
	kafkaClient  kafka.Client
	cfg          *config.Config
	otel         otel.Otel
}

func New(scheduleRepo scheduleRepo.Schedule, movieRepo movieRepo.Movie, kafkaClient kafka.Client, cfg *config.Config, otel otel.Otel) Schedule {
	return &serviceImpl{
		scheduleRepo: scheduleRepo,
		movieRepo:    movieRepo,
		kafkaClient:  kafkaClient,
		cfg:          cfg,
		otel:         otel,
	}
}

// Cancel marks a schedule cancelled and publishes schedule.cancelled, which the
// refund consumer fans out into a refund per confirmed booking.
func (s *serviceImpl) Cancel(ctx context.Context, id, userID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.Cancel")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.scheduleRepo.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	if existing.ID == "" {
		return failure.NotFound("schedule not found")
	}

	if existing.Status == model.StatusCancelled {
		return nil
	}

	update := map[string]any{
		model.FieldStatus:        model.StatusCancelled,
		constant.FieldModifiedAt: timezone.Now(),
		constant.FieldModifiedBy: userID,
	}

	if err = s.scheduleRepo.Update(ctx, update, filter); err != nil {
		return fmt.Errorf("failed to cancel schedule: %w", err)
	}

	if s.cfg.Kafka.Enable {
		if pErr := s.kafkaClient.SendMessages(ctx, constant.TopicScheduleCancelled, kafka.Message{
			Key:   id,
			Value: map[string]any{"schedule_id": id},
		}); pErr != nil {
			log.Error().Err(pErr).Msg("failed to publish schedule.cancelled")
		}
	}

	return nil
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateScheduleRequest, userID string) (res dto.ScheduleResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	movie, err := s.movieRepo.Get(ctx, shared.FilterByID(req.MovieID, movieModel.FieldID, movieModel.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get movie: %w", err)
	}

	if movie.ID == "" {
		return res, failure.BadRequestFromString("movie not found")
	}

	schedule := req.ToModel(userID, movie.DurationMin)

	if err = s.scheduleRepo.Insert(ctx, schedule); err != nil {
		if mapped := mapScheduleConstraintErr(err); mapped != nil {
			err = mapped

			return res, err
		}

		log.Error().Err(err).Msg("failed to insert schedule")

		return res, fmt.Errorf("failed to insert schedule: %w", err)
	}

	res.FromModel(schedule)

	return res, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (res dto.ScheduleResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.GetByID")
	defer scope.End()
	defer scope.TraceIfError(err)

	schedule, err := s.scheduleRepo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule.ID == "" {
		return res, failure.NotFound("schedule not found")
	}

	res.FromModel(schedule)

	return res, nil
}

func (s *serviceImpl) GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetSchedulesResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	total, err := s.scheduleRepo.Count(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to count schedules: %w", err)
	}

	schedules, err := s.scheduleRepo.GetAll(ctx, params, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get schedules: %w", err)
	}

	res.FromModels(schedules, total, params.Limit)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, id string, req dto.UpdateScheduleRequest, userID string) (res dto.ScheduleResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.scheduleRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get schedule: %w", err)
	}

	if existing.ID == "" {
		return res, failure.NotFound("schedule not found")
	}

	updatedFields := map[string]any{
		constant.FieldModifiedAt: timezone.Now(),
		constant.FieldModifiedBy: userID,
	}

	if req.Price != nil {
		updatedFields[model.FieldPrice] = *req.Price
	}

	if req.Status != nil {
		if !constant.ScheduleStatus(*req.Status).Valid() {
			return res, failure.BadRequestFromString("invalid schedule status")
		}

		updatedFields[model.FieldStatus] = *req.Status
	}

	// Changing start_time recomputes end_time and show_date from the movie duration.
	if req.StartTime != nil {
		movie, mErr := s.movieRepo.Get(ctx, shared.FilterByID(existing.MovieID, movieModel.FieldID, movieModel.TableName))
		if mErr != nil {
			return res, fmt.Errorf("failed to get movie: %w", mErr)
		}

		start := *req.StartTime
		y, m, d := start.Date()
		updatedFields[model.FieldStartTime] = start
		updatedFields[model.FieldEndTime] = start.Add(time.Duration(movie.DurationMin) * time.Minute)
		updatedFields[model.FieldShowDate] = time.Date(y, m, d, 0, 0, 0, 0, start.Location())
	}

	if err = s.scheduleRepo.Update(ctx, updatedFields, filter); err != nil {
		if mapped := mapScheduleConstraintErr(err); mapped != nil {
			err = mapped

			return res, err
		}

		return res, fmt.Errorf("failed to update schedule: %w", err)
	}

	updated, err := s.scheduleRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get updated schedule: %w", err)
	}

	res.FromModel(updated)

	return res, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".schedule.Delete")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.scheduleRepo.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	if existing.ID == "" {
		return failure.NotFound("schedule not found")
	}

	if err = s.scheduleRepo.Delete(ctx, filter); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == constant.PqErrorCodeFkViolation {
			return failure.Conflict("schedule has bookings and cannot be deleted; cancel it instead")
		}

		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// mapScheduleConstraintErr translates Postgres constraint violations on the
// schedules table into clean client errors. Returns nil if not a known one.
func mapScheduleConstraintErr(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return nil
	}

	switch string(pqErr.Code) {
	case constant.PqErrorCodeExclusionViolation:
		return failure.Conflict("the studio already has a schedule overlapping this time")
	case constant.PqErrorCodeFkViolation:
		return failure.BadRequestFromString("studio or movie not found")
	case constant.PqErrorCodeCheckViolation:
		return failure.BadRequestFromString("invalid schedule time range")
	default:
		return nil
	}
}
