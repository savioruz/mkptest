package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/internal/domains/studio/model"
	gDto "oil/shared/dto"
	gRepo "oil/shared/repository"

	"github.com/jmoiron/sqlx"
)

type Studio interface {
	Insert(ctx context.Context, model model.Studio) error
	InsertTx(ctx context.Context, tx *sqlx.Tx, model model.Studio) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Studio, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Studio, error)
	Exist(ctx context.Context, filter gDto.FilterGroup) (bool, error)
	Count(ctx context.Context, filter gDto.FilterGroup) (int, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	Delete(ctx context.Context, filter gDto.FilterGroup) error
}

type repositoryImpl struct {
	gRepo.Repository[model.Studio]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Studio {
	return &repositoryImpl{
		Repository: gRepo.NewRepository[model.Studio](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
