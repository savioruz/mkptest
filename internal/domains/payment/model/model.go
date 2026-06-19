package model

import (
	"time"

	"oil/shared/constant"
	"oil/shared/model"
)

const (
	TableName  = "payments"
	EntityName = "payment"

	FieldID             = "id"
	FieldBookingID      = "booking_id"
	FieldAmount         = "amount"
	FieldStatus         = "status"
	FieldReference      = "reference"
	FieldIdempotencyKey = "idempotency_key"
	FieldMethod         = "method"
	FieldPaidAt         = "paid_at"

	StatusPending = string(constant.PaymentStatusPending)
	StatusSuccess = string(constant.PaymentStatusSuccess)
	StatusFailed  = string(constant.PaymentStatusFailed)
)

type Payment struct {
	ID             string     `db:"id"`
	BookingID      string     `db:"booking_id"`
	Amount         float64    `db:"amount"`
	Status         string     `db:"status"`
	Reference      string     `db:"reference"`
	IdempotencyKey string     `db:"idempotency_key"`
	Method         *string    `db:"method"`
	PaidAt         *time.Time `db:"paid_at"`
	model.Metadata
}
