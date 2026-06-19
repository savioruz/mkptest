package refund

import (
	"encoding/json"
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/refund/service"
	"oil/shared/constant"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Refund
	otel    otel.Otel
}

func New(service service.Refund, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	// Customer self-service refund for one of their own confirmed bookings.
	r.Post("/bookings/{id}/refund", handler.SelfRefund)
}

type selfRefundRequest struct {
	Reason string `json:"reason"`
}

// SelfRefund lets a customer request a refund for their confirmed booking
// @Summary Request a refund (customer)
// @Description Request a refund for one of your confirmed bookings. Processed asynchronously.
// @Tags Refund
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Param request body selfRefundRequest false "Refund reason"
// @Success 202 {object} response.Message
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/bookings/{id}/refund [post]
func (handler *Handler) SelfRefund(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".refund.SelfRefund")
	defer scope.End()

	bookingID := chi.URLParam(r, constant.RequestParamID)
	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	req := selfRefundRequest{}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if _, err := handler.service.RequestForBooking(ctx, bookingID, userID, req.Reason); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to request refund")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusAccepted, "refund requested; it is being processed")
}
