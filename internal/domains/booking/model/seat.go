package model

import (
	"oil/shared/constant"
	"oil/shared/model"
)

const (
	SeatTableName  = "booking_seats"
	SeatEntityName = "booking_seat"

	SeatFieldID         = "id"
	SeatFieldBookingID  = "booking_id"
	SeatFieldScheduleID = "schedule_id"
	SeatFieldSeatID     = "seat_id"
	SeatFieldStatus     = "status"
	SeatFieldPrice      = "price"

	SeatStatusHeld     = string(constant.SeatBookingStatusHeld)
	SeatStatusBooked   = string(constant.SeatBookingStatusBooked)
	SeatStatusReleased = string(constant.SeatBookingStatusReleased)
)

type BookingSeat struct {
	ID         string  `db:"id"`
	BookingID  string  `db:"booking_id"`
	ScheduleID string  `db:"schedule_id"`
	SeatID     string  `db:"seat_id"`
	Status     string  `db:"status"`
	Price      float64 `db:"price"`
	model.Metadata
}
