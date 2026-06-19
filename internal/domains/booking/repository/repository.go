package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/internal/domains/booking/model"
	gDto "oil/shared/dto"
	gRepo "oil/shared/repository"

	"github.com/jmoiron/sqlx"
)

// Booking is the repository for the bookings table.
type Booking interface {
	Insert(ctx context.Context, model model.Booking) error
	InsertTx(ctx context.Context, tx *sqlx.Tx, model model.Booking) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Booking, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Booking, error)
	Count(ctx context.Context, filter gDto.FilterGroup) (int, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, req map[string]any, filter gDto.FilterGroup) error
}

type bookingRepositoryImpl struct {
	gRepo.Repository[model.Booking]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Booking {
	return &bookingRepositoryImpl{
		Repository: gRepo.NewRepository[model.Booking](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}

// Seat is the repository for the booking_seats table.
type Seat interface {
	InsertBulkTx(ctx context.Context, tx *sqlx.Tx, models []model.BookingSeat) error
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.BookingSeat, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, req map[string]any, filter gDto.FilterGroup) error
}

type seatRepositoryImpl struct {
	gRepo.Repository[model.BookingSeat]
	db   *postgres.Connection
	otel otel.Otel
}

func NewSeat(db *postgres.Connection, otel otel.Otel) Seat {
	return &seatRepositoryImpl{
		Repository: gRepo.NewRepository[model.BookingSeat](model.SeatEntityName, model.SeatTableName, model.SeatFieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
