package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/internal/domains/seat/model"
	gDto "oil/shared/dto"
	gRepo "oil/shared/repository"

	"github.com/jmoiron/sqlx"
)

type Seat interface {
	Insert(ctx context.Context, model model.Seat) error
	InsertBulk(ctx context.Context, models []model.Seat) error
	InsertBulkTx(ctx context.Context, tx *sqlx.Tx, models []model.Seat) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Seat, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Seat, error)
	Exist(ctx context.Context, filter gDto.FilterGroup) (bool, error)
	Count(ctx context.Context, filter gDto.FilterGroup) (int, error)
	Delete(ctx context.Context, filter gDto.FilterGroup) error
}

type repositoryImpl struct {
	gRepo.Repository[model.Seat]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Seat {
	return &repositoryImpl{
		Repository: gRepo.NewRepository[model.Seat](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
