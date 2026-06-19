package booking

import (
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/booking/model/dto"
	"oil/internal/domains/booking/service"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Booking
	otel    otel.Otel
}

func New(service service.Booking, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/bookings", func(r chi.Router) {
		r.Get("/", handler.GetUserBookings)
		r.Post("/", handler.Create)
		r.Get("/{id}", handler.GetByID)
	})

	// Seat map is schedule-scoped but served by the booking domain (it blends
	// seats, booked rows and live Redis holds).
	r.Get("/schedules/{id}/seat-map", handler.SeatMap)
}

func isAdmin(role string) bool {
	return role == constant.RoleAdmin || role == constant.RoleSuperAdmin
}

// Create handles seat holding + pending booking creation
// @Summary Create a booking (hold seats)
// @Description Hold the selected seats and create a PENDING booking with a payment reference. Seats are held for the configured window.
// @Tags Booking
// @Accept json
// @Produce json
// @Param request body dto.CreateBookingRequest true "Create Booking Request"
// @Success 201 {object} dto.BookingResponse
// @Failure 400 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/bookings [post]
func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".booking.Create")
	defer scope.End()

	req := dto.CreateBookingRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")
		response.WithError(w, err)

		return
	}

	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	res, err := handler.service.Create(ctx, userID, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create booking")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusCreated, res)
}

// GetUserBookings lists the authenticated user's bookings
// @Summary List my bookings
// @Tags Booking
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} dto.GetBookingsResponse
// @Security BearerAuth
// @Router /api/bookings [get]
func (handler *Handler) GetUserBookings(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".booking.GetUserBookings")
	defer scope.End()

	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	params := gDto.QueryParams{}
	params.FromRequest(r, true)

	res, err := handler.service.GetUserBookings(ctx, userID, params)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get bookings")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// GetByID returns a booking with its seats
// @Summary Get a booking
// @Tags Booking
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} dto.BookingResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/bookings/{id} [get]
func (handler *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".booking.GetByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)
	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)
	role, _ := ctx.Value(constant.ContextKeyUserRole).(string)

	res, err := handler.service.GetByID(ctx, id, userID, isAdmin(role))
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get booking")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// SeatMap returns the live seat availability for a schedule
// @Summary Get schedule seat map
// @Description Seat availability = total seats − active holds − booked seats.
// @Tags Booking
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} dto.SeatMapResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules/{id}/seat-map [get]
func (handler *Handler) SeatMap(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".booking.SeatMap")
	defer scope.End()

	scheduleID := chi.URLParam(r, constant.RequestParamID)

	res, err := handler.service.SeatMap(ctx, scheduleID)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get seat map")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}
