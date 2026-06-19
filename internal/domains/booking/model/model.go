package model

import (
	"time"

	"oil/shared/constant"
	"oil/shared/model"
)

const (
	TableName  = "bookings"
	EntityName = "booking"

	FieldID          = "id"
	FieldBookingCode = "booking_code"
	FieldUserID      = "user_id"
	FieldScheduleID  = "schedule_id"
	FieldStatus      = "status"
	FieldTotalAmount = "total_amount"
	FieldSeatCount   = "seat_count"
	FieldExpiresAt   = "expires_at"
	FieldActive      = "active"

	StatusPending   = string(constant.BookingStatusPending)
	StatusConfirmed = string(constant.BookingStatusConfirmed)
	StatusExpired   = string(constant.BookingStatusExpired)
	StatusCancelled = string(constant.BookingStatusCancelled)
	StatusRefunded  = string(constant.BookingStatusRefunded)
)

type Booking struct {
	ID          string     `db:"id"`
	BookingCode string     `db:"booking_code"`
	UserID      string     `db:"user_id"`
	ScheduleID  string     `db:"schedule_id"`
	Status      string     `db:"status"`
	TotalAmount float64    `db:"total_amount"`
	SeatCount   int        `db:"seat_count"`
	ExpiresAt   *time.Time `db:"expires_at"`
	Active      bool       `db:"active"`
	model.Metadata
}
