package model

import "oil/shared/model"

const (
	TableName  = "cinemas"
	EntityName = "cinema"

	FieldID      = "id"
	FieldName    = "name"
	FieldCity    = "city"
	FieldAddress = "address"
	FieldActive  = "active"
)

type Cinema struct {
	ID      string  `db:"id"`
	Name    string  `db:"name"`
	City    string  `db:"city"`
	Address *string `db:"address"`
	Active  bool    `db:"active"`
	model.Metadata
}
