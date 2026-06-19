package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"

	"oil/infras/otel"
	"oil/infras/postgres"
	seatModel "oil/internal/domains/seat/model"
	seatRepo "oil/internal/domains/seat/repository"
	"oil/internal/domains/studio/model"
	"oil/internal/domains/studio/model/dto"
	studioRepo "oil/internal/domains/studio/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Studio interface {
	Create(ctx context.Context, req dto.CreateStudioRequest, userID string) (dto.StudioResponse, error)
	GetByID(ctx context.Context, id string) (dto.StudioResponse, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (dto.GetStudiosResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateStudioRequest, userID string) (dto.StudioResponse, error)
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	studioRepo studioRepo.Studio
	seatRepo   seatRepo.Seat
	db         *postgres.Connection
	otel       otel.Otel
}

func New(studioRepo studioRepo.Studio, seatRepo seatRepo.Seat, db *postgres.Connection, otel otel.Otel) Studio {
	return &serviceImpl{
		studioRepo: studioRepo,
		seatRepo:   seatRepo,
		db:         db,
		otel:       otel,
	}
}

// Create inserts the studio and auto-generates its seat grid atomically:
// either both the studio row and all seat rows are committed, or neither.
func (s *serviceImpl) Create(ctx context.Context, req dto.CreateStudioRequest, userID string) (res dto.StudioResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".studio.Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	studio := req.ToModel(userID)
	seats := buildSeats(studio, req.VipRows, userID)

	tx, err := s.db.Write.BeginTxx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.studioRepo.InsertTx(ctx, tx, studio); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch string(pqErr.Code) {
			case constant.PqErrorCodeFkViolation:
				return res, failure.BadRequestFromString("cinema not found")
			case constant.PqErrorCodeUniqueViolation:
				return res, failure.Conflict("a studio with this name already exists in the cinema")
			}
		}

		log.Error().Err(err).Msg("failed to insert studio")

		return res, fmt.Errorf("failed to insert studio: %w", err)
	}

	if err = s.seatRepo.InsertBulkTx(ctx, tx, seats); err != nil {
		log.Error().Err(err).Msg("failed to insert seats")

		return res, fmt.Errorf("failed to insert seats: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return res, fmt.Errorf("failed to commit transaction: %w", err)
	}

	res.FromModel(studio)

	return res, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (res dto.StudioResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".studio.GetByID")
	defer scope.End()
	defer scope.TraceIfError(err)

	studio, err := s.studioRepo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get studio: %w", err)
	}

	if studio.ID == "" {
		return res, failure.NotFound("studio not found")
	}

	res.FromModel(studio)

	return res, nil
}

func (s *serviceImpl) GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetStudiosResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".studio.GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	total, err := s.studioRepo.Count(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to count studios: %w", err)
	}

	studios, err := s.studioRepo.GetAll(ctx, params, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get studios: %w", err)
	}

	res.FromModels(studios, total, params.Limit)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, id string, req dto.UpdateStudioRequest, userID string) (res dto.StudioResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".studio.Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.studioRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get studio: %w", err)
	}

	if existing.ID == "" {
		return res, failure.NotFound("studio not found")
	}

	updatedFields := shared.TransformFields(req, userID)
	if err = s.studioRepo.Update(ctx, updatedFields, filter); err != nil {
		return res, fmt.Errorf("failed to update studio: %w", err)
	}

	updated, err := s.studioRepo.Get(ctx, filter)
	if err != nil {
		return res, fmt.Errorf("failed to get updated studio: %w", err)
	}

	res.FromModel(updated)

	return res, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".studio.Delete")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	existing, err := s.studioRepo.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get studio: %w", err)
	}

	if existing.ID == "" {
		return failure.NotFound("studio not found")
	}

	if err = s.studioRepo.Delete(ctx, filter); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == constant.PqErrorCodeFkViolation {
			return failure.Conflict("studio is still referenced by " + constant.FkUsed)
		}

		return fmt.Errorf("failed to delete studio: %w", err)
	}

	return nil
}

// buildSeats generates the seat grid for a studio: rows A, B, C... and columns
// 1..ColsPerRow. The last VipRows rows (if any) are marked as VIP seating.
func buildSeats(studio model.Studio, vipRows int, username string) []seatModel.Seat {
	now := timezone.Now()
	seats := make([]seatModel.Seat, 0, studio.RowCount*studio.ColsPerRow)

	for ri := 0; ri < studio.RowCount; ri++ {
		rowLetter := string(rune('A' + ri))
		seatType := seatModel.SeatTypeRegular

		if vipRows > 0 && ri >= studio.RowCount-vipRows {
			seatType = seatModel.SeatTypeVIP
		}

		for ci := 1; ci <= studio.ColsPerRow; ci++ {
			rl := rowLetter
			sn := ci

			seats = append(seats, seatModel.Seat{
				ID:         uuid.NewString(),
				StudioID:   studio.ID,
				SeatLabel:  fmt.Sprintf("%s%d", rowLetter, ci),
				RowLabel:   &rl,
				SeatNumber: &sn,
				SeatType:   seatType,
				Active:     true,
				Metadata: gModel.Metadata{
					CreatedAt:  now,
					ModifiedAt: now,
					CreatedBy:  username,
					ModifiedBy: username,
				},
			})
		}
	}

	return seats
}
