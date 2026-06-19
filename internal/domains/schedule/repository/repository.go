package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/internal/domains/schedule/model"
	gDto "oil/shared/dto"
	gRepo "oil/shared/repository"
)

type Schedule interface {
	Insert(ctx context.Context, model model.Schedule) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Schedule, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Schedule, error)
	Exist(ctx context.Context, filter gDto.FilterGroup) (bool, error)
	Count(ctx context.Context, filter gDto.FilterGroup) (int, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	Delete(ctx context.Context, filter gDto.FilterGroup) error
}

type repositoryImpl struct {
	gRepo.Repository[model.Schedule]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Schedule {
	return &repositoryImpl{
		Repository: gRepo.NewRepository[model.Schedule](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
