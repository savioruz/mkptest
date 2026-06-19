package model

import (
	"time"

	"oil/shared/constant"
	"oil/shared/model"
)

const (
	TableName  = "schedules"
	EntityName = "schedule"

	FieldID        = "id"
	FieldMovieID   = "movie_id"
	FieldStudioID  = "studio_id"
	FieldShowDate  = "show_date"
	FieldStartTime = "start_time"
	FieldEndTime   = "end_time"
	FieldPrice     = "price"
	FieldStatus    = "status"
	FieldActive    = "active"

	StatusScheduled = string(constant.ScheduleStatusScheduled)
	StatusCancelled = string(constant.ScheduleStatusCancelled)
	StatusFinished  = string(constant.ScheduleStatusFinished)
)

type Schedule struct {
	ID        string    `db:"id"`
	MovieID   string    `db:"movie_id"`
	StudioID  string    `db:"studio_id"`
	ShowDate  time.Time `db:"show_date"`
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
	Price     float64   `db:"price"`
	Status    string    `db:"status"`
	Active    bool      `db:"active"`
	model.Metadata
}
