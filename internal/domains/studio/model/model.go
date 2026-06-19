package model

import "oil/shared/model"

const (
	TableName  = "studios"
	EntityName = "studio"

	FieldID         = "id"
	FieldCinemaID   = "cinema_id"
	FieldName       = "name"
	FieldTotalSeats = "total_seats"
	FieldRowCount   = "row_count"
	FieldColsPerRow = "cols_per_row"
	FieldActive     = "active"
)

type Studio struct {
	ID         string `db:"id"`
	CinemaID   string `db:"cinema_id"`
	Name       string `db:"name"`
	TotalSeats int    `db:"total_seats"`
	RowCount   int    `db:"row_count"`
	ColsPerRow int    `db:"cols_per_row"`
	Active     bool   `db:"active"`
	model.Metadata
}
