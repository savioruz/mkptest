package dto

import (
	"oil/internal/domains/booking/model"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
)

type CreateBookingRequest struct {
	ScheduleID string   `json:"schedule_id" validate:"required"`
	SeatIDs    []string `json:"seat_ids"    validate:"required,min=1,dive,required"`
}

type BookingSeatItem struct {
	SeatID string  `json:"seat_id"`
	Status string  `json:"status"`
	Price  float64 `json:"price"`
}

type BookingResponse struct {
	ID               string            `json:"id"`
	BookingCode      string            `json:"booking_code"`
	UserID           string            `json:"user_id"`
	ScheduleID       string            `json:"schedule_id"`
	Status           string            `json:"status"`
	TotalAmount      float64           `json:"total_amount"`
	SeatCount        int               `json:"seat_count"`
	ExpiresAt        string            `json:"expires_at,omitempty"`
	PaymentReference string            `json:"payment_reference,omitempty"`
	Seats            []BookingSeatItem `json:"seats,omitempty"`
	gDto.Metadata
}

func (r *BookingResponse) FromModel(m model.Booking) {
	r.ID = m.ID
	r.BookingCode = m.BookingCode
	r.UserID = m.UserID
	r.ScheduleID = m.ScheduleID
	r.Status = m.Status
	r.TotalAmount = m.TotalAmount
	r.SeatCount = m.SeatCount

	if m.ExpiresAt != nil {
		r.ExpiresAt = m.ExpiresAt.Format(constant.DateFormat)
	}

	r.Metadata.FromModel(m.Metadata)
}

func (r *BookingResponse) SetSeats(seats []model.BookingSeat) {
	r.Seats = make([]BookingSeatItem, len(seats))
	for i, s := range seats {
		r.Seats[i] = BookingSeatItem{SeatID: s.SeatID, Status: s.Status, Price: s.Price}
	}
}

type GetBookingsResponse struct {
	Bookings  []BookingResponse `json:"bookings"`
	TotalPage int               `json:"total_page"`
	TotalData int               `json:"total_data"`
}

func (r *GetBookingsResponse) FromModels(models []model.Booking, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Bookings = make([]BookingResponse, len(models))
	for i, mod := range models {
		r.Bookings[i].FromModel(mod)
	}
}

// --- Seat map ---

type SeatMapSeat struct {
	SeatID    string `json:"seat_id"`
	SeatLabel string `json:"seat_label"`
	RowLabel  string `json:"row_label,omitempty"`
	SeatType  string `json:"seat_type"`
	Status    string `json:"status"` // available | held | booked
}

type SeatMapResponse struct {
	ScheduleID string        `json:"schedule_id"`
	TotalSeats int           `json:"total_seats"`
	Available  int           `json:"available"`
	Held       int           `json:"held"`
	Booked     int           `json:"booked"`
	Seats      []SeatMapSeat `json:"seats"`
}
