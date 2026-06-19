package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"

	"oil/infras/otel"
	"oil/internal/domains/cinema/model"
	"oil/internal/domains/cinema/model/dto"
	cinemaRepo "oil/internal/domains/cinema/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Cinema interface {
	Create(ctx context.Context, req dto.CreateCinemaRequest, userID string) (dto.CinemaResponse, error)
	GetByID(ctx context.Context, id string) (dto.CinemaResponse, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (dto.GetCinemasResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateCinemaRequest, userID string) (dto.CinemaResponse, error)
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	cinemaRepo cinemaRepo.Cinema
	otel       otel.Otel
}

func New(cinemaRepo cinemaRepo.Cinema, otel otel.Otel) Cinema {
	return &serviceImpl{
		cinemaRepo: cinemaRepo,
		otel:       otel,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateCinemaRequest, userID string) (res dto.CinemaResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".cinema.Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	cinema := req.ToModel(userID)
	if err = s.cinemaRepo.Insert(ctx, cinema); err != nil {
		log.Error().Err(err).Msg("failed to insert cinema")

		return res, fmt.Errorf("failed to insert cinema: %w", err)
	}

	res.FromModel(cinema)

	return res, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (res dto.CinemaResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".cinema.GetByID")
	defer scope.End()
	defer scope.TraceIfError(err)

	cinema, err := s.cinemaRepo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get cinema")

		return res, fmt.Errorf("failed to get cinema: %w", err)
	}

	if cinema.ID == "" {
		return res, failure.NotFound("cinema not found")
	}

	res.FromModel(cinema)

	return res, nil
}

func (s *serviceImpl) GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetCinemasResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".cinema.GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	total, err := s.cinemaRepo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count cinemas")

		return res, fmt.Errorf("failed to count cinemas: %w", err)
	}

	cinemas, err := s.cinemaRepo.GetAll(ctx, params, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get cinemas")

		return res, fmt.Errorf("failed to get cinemas: %w", err)
	}

	res.FromModels(cinemas, total, params.Limit)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, id string, req dto.UpdateCinemaRequest, userID string) (res dto.CinemaResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".cinema.Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.cinemaRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get cinema: %w", err)
	}

	if existing.ID == "" {
		return res, failure.NotFound("cinema not found")
	}

	updatedFields := shared.TransformFields(req, userID)
	if err = s.cinemaRepo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update cinema")

		return res, fmt.Errorf("failed to update cinema: %w", err)
	}

	updated, err := s.cinemaRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get updated cinema: %w", err)
	}

	res.FromModel(updated)

	return res, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".cinema.Delete")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.cinemaRepo.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get cinema: %w", err)
	}

	if existing.ID == "" {
		return failure.NotFound("cinema not found")
	}

	if err = s.cinemaRepo.Delete(ctx, filter); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == constant.PqErrorCodeFkViolation {
			return failure.Conflict("cinema is still referenced by " + constant.FkUsed)
		}

		log.Error().Err(err).Msg("failed to delete cinema")

		return fmt.Errorf("failed to delete cinema: %w", err)
	}

	return nil
}
