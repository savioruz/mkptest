package model

import "oil/shared/model"

const (
	TableName  = "movies"
	EntityName = "movie"

	FieldID          = "id"
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldDurationMin = "duration_min"
	FieldGenre       = "genre"
	FieldRating      = "rating"
	FieldPosterURL   = "poster_url"
	FieldActive      = "active"
)

type Movie struct {
	ID          string  `db:"id"`
	Title       string  `db:"title"`
	Description *string `db:"description"`
	DurationMin int     `db:"duration_min"`
	Genre       *string `db:"genre"`
	Rating      *string `db:"rating"`
	PosterURL   *string `db:"poster_url"`
	Active      bool    `db:"active"`
	model.Metadata
}
