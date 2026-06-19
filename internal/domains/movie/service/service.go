package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"

	"oil/infras/otel"
	"oil/internal/domains/movie/model"
	"oil/internal/domains/movie/model/dto"
	movieRepo "oil/internal/domains/movie/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Movie interface {
	Create(ctx context.Context, req dto.CreateMovieRequest, userID string) (dto.MovieResponse, error)
	GetByID(ctx context.Context, id string) (dto.MovieResponse, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (dto.GetMoviesResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateMovieRequest, userID string) (dto.MovieResponse, error)
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	movieRepo movieRepo.Movie
	otel      otel.Otel
}

func New(movieRepo movieRepo.Movie, otel otel.Otel) Movie {
	return &serviceImpl{
		movieRepo: movieRepo,
		otel:      otel,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateMovieRequest, userID string) (res dto.MovieResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".movie.Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	movie := req.ToModel(userID)
	if err = s.movieRepo.Insert(ctx, movie); err != nil {
		log.Error().Err(err).Msg("failed to insert movie")

		return res, fmt.Errorf("failed to insert movie: %w", err)
	}

	res.FromModel(movie)

	return res, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (res dto.MovieResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".movie.GetByID")
	defer scope.End()
	defer scope.TraceIfError(err)

	movie, err := s.movieRepo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get movie")

		return res, fmt.Errorf("failed to get movie: %w", err)
	}

	if movie.ID == "" {
		return res, failure.NotFound("movie not found")
	}

	res.FromModel(movie)

	return res, nil
}

func (s *serviceImpl) GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetMoviesResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".movie.GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	total, err := s.movieRepo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count movies")

		return res, fmt.Errorf("failed to count movies: %w", err)
	}

	movies, err := s.movieRepo.GetAll(ctx, params, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get movies")

		return res, fmt.Errorf("failed to get movies: %w", err)
	}

	res.FromModels(movies, total, params.Limit)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, id string, req dto.UpdateMovieRequest, userID string) (res dto.MovieResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".movie.Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.movieRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get movie: %w", err)
	}

	if existing.ID == "" {
		return res, failure.NotFound("movie not found")
	}

	updatedFields := shared.TransformFields(req, userID)
	if err = s.movieRepo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update movie")

		return res, fmt.Errorf("failed to update movie: %w", err)
	}

	updated, err := s.movieRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get updated movie: %w", err)
	}

	res.FromModel(updated)

	return res, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".movie.Delete")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.movieRepo.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get movie: %w", err)
	}

	if existing.ID == "" {
		return failure.NotFound("movie not found")
	}

	if err = s.movieRepo.Delete(ctx, filter); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == constant.PqErrorCodeFkViolation {
			return failure.Conflict("movie is still referenced by " + constant.FkUsed)
		}

		log.Error().Err(err).Msg("failed to delete movie")

		return fmt.Errorf("failed to delete movie: %w", err)
	}

	return nil
}
