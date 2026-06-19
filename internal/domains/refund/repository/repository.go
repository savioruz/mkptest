package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/internal/domains/refund/model"
	gDto "oil/shared/dto"
	gRepo "oil/shared/repository"

	"github.com/jmoiron/sqlx"
)

type Refund interface {
	Insert(ctx context.Context, model model.Refund) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Refund, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Refund, error)
	Exist(ctx context.Context, filter gDto.FilterGroup) (bool, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, req map[string]any, filter gDto.FilterGroup) error
}

type repositoryImpl struct {
	gRepo.Repository[model.Refund]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Refund {
	return &repositoryImpl{
		Repository: gRepo.NewRepository[model.Refund](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
