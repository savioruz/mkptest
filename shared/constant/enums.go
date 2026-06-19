package constant

// Domain enumerations. These replace the DB CHECK (... IN ...) constraints that
// were removed from the migrations: the set of allowed values now lives here as
// typed constants, and the services validate against them via Valid().

// SeatType is the class of a physical seat.
type SeatType string

const (
	SeatTypeRegular SeatType = "regular"
	SeatTypeVIP     SeatType = "vip"
)

func (s SeatType) Valid() bool {
	switch s {
	case SeatTypeRegular, SeatTypeVIP:
		return true
	default:
		return false
	}
}

// ScheduleStatus is the lifecycle state of a show schedule.
type ScheduleStatus string

const (
	ScheduleStatusScheduled ScheduleStatus = "scheduled"
	ScheduleStatusCancelled ScheduleStatus = "cancelled"
	ScheduleStatusFinished  ScheduleStatus = "finished"
)

func (s ScheduleStatus) Valid() bool {
	switch s {
	case ScheduleStatusScheduled, ScheduleStatusCancelled, ScheduleStatusFinished:
		return true
	default:
		return false
	}
}

// BookingStatus is the lifecycle state of a booking.
type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusExpired   BookingStatus = "expired"
	BookingStatusCancelled BookingStatus = "cancelled"
	BookingStatusRefunded  BookingStatus = "refunded"
)

func (s BookingStatus) Valid() bool {
	switch s {
	case BookingStatusPending, BookingStatusConfirmed, BookingStatusExpired, BookingStatusCancelled, BookingStatusRefunded:
		return true
	default:
		return false
	}
}

// SeatBookingStatus is the state of a seat within a booking (booking_seats).
type SeatBookingStatus string

const (
	SeatBookingStatusHeld     SeatBookingStatus = "held"
	SeatBookingStatusBooked   SeatBookingStatus = "booked"
	SeatBookingStatusReleased SeatBookingStatus = "released"
)

func (s SeatBookingStatus) Valid() bool {
	switch s {
	case SeatBookingStatusHeld, SeatBookingStatusBooked, SeatBookingStatusReleased:
		return true
	default:
		return false
	}
}

// PaymentStatus is the settlement state of a payment.
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusSuccess, PaymentStatusFailed:
		return true
	default:
		return false
	}
}

// RefundStatus is the processing state of a refund.
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusCompleted  RefundStatus = "completed"
	RefundStatusFailed     RefundStatus = "failed"
)

func (s RefundStatus) Valid() bool {
	switch s {
	case RefundStatusPending, RefundStatusProcessing, RefundStatusCompleted, RefundStatusFailed:
		return true
	default:
		return false
	}
}

// RefundInitiator is who initiated a refund.
type RefundInitiator string

const (
	RefundInitiatedByCinema   RefundInitiator = "cinema"
	RefundInitiatedByCustomer RefundInitiator = "customer"
)

func (r RefundInitiator) Valid() bool {
	switch r {
	case RefundInitiatedByCinema, RefundInitiatedByCustomer:
		return true
	default:
		return false
	}
}
