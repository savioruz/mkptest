package model

import (
	"time"

	"oil/shared/constant"
	"oil/shared/model"
)

const (
	TableName  = "refunds"
	EntityName = "refund"

	FieldID             = "id"
	FieldBookingID      = "booking_id"
	FieldPaymentID      = "payment_id"
	FieldAmount         = "amount"
	FieldStatus         = "status"
	FieldInitiatedBy    = "initiated_by"
	FieldIdempotencyKey = "idempotency_key"
	FieldReason         = "reason"
	FieldRefundedAt     = "refunded_at"

	// Status / initiator values are sourced from shared/constant (single source
	// of truth; the DB CHECK constraints were removed in favour of service-side
	// validation).
	StatusPending    = string(constant.RefundStatusPending)
	StatusProcessing = string(constant.RefundStatusProcessing)
	StatusCompleted  = string(constant.RefundStatusCompleted)
	StatusFailed     = string(constant.RefundStatusFailed)

	InitiatedByCinema   = string(constant.RefundInitiatedByCinema)
	InitiatedByCustomer = string(constant.RefundInitiatedByCustomer)
)

type Refund struct {
	ID             string     `db:"id"`
	BookingID      string     `db:"booking_id"`
	PaymentID      *string    `db:"payment_id"`
	Amount         float64    `db:"amount"`
	Status         string     `db:"status"`
	InitiatedBy    string     `db:"initiated_by"`
	IdempotencyKey string     `db:"idempotency_key"`
	Reason         *string    `db:"reason"`
	RefundedAt     *time.Time `db:"refunded_at"`
	model.Metadata
}
