package model

import (
	"oil/shared/constant"
	"oil/shared/model"
)

const (
	TableName  = "seats"
	EntityName = "seat"

	FieldID         = "id"
	FieldStudioID   = "studio_id"
	FieldSeatLabel  = "seat_label"
	FieldRowLabel   = "row_label"
	FieldSeatNumber = "seat_number"
	FieldSeatType   = "seat_type"
	FieldActive     = "active"

	SeatTypeRegular = string(constant.SeatTypeRegular)
	SeatTypeVIP     = string(constant.SeatTypeVIP)
)

type Seat struct {
	ID         string  `db:"id"`
	StudioID   string  `db:"studio_id"`
	SeatLabel  string  `db:"seat_label"`
	RowLabel   *string `db:"row_label"`
	SeatNumber *int    `db:"seat_number"`
	SeatType   string  `db:"seat_type"`
	Active     bool    `db:"active"`
	model.Metadata
}
