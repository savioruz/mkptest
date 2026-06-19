package payment

import (
	"context"
	"net/http"

	"oil/config"
	"oil/infras/otel"
	bookingSvc "oil/internal/domains/booking/service"
	"oil/internal/domains/payment/model/dto"
	paymentSvc "oil/internal/domains/payment/service"
	"oil/shared/constant"
	"oil/shared/failure"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const webhookSecretHeader = "X-Webhook-Secret"

type Handler struct {
	paymentService paymentSvc.Payment
	bookingService bookingSvc.Booking
	cfg            *config.Config
	otel           otel.Otel
}

func New(paymentService paymentSvc.Payment, bookingService bookingSvc.Booking, cfg *config.Config, otel otel.Otel) Handler {
	return Handler{
		paymentService: paymentService,
		bookingService: bookingService,
		cfg:            cfg,
		otel:           otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/payments", func(r chi.Router) {
		r.Post("/webhook", handler.Webhook)
	})
}

// Webhook settles a charge. Reviewers POST this to simulate the gateway:
// success -> confirm booking; failed -> release booking. Guarded by a shared
// secret header (the route skips JWT auth). Idempotent on repeated delivery.
// @Summary Payment gateway webhook (mock)
// @Description Settle a payment. Header X-Webhook-Secret must match PAYMENT_WEBHOOK_SECRET.
// @Tags Payment
// @Accept json
// @Produce json
// @Param X-Webhook-Secret header string true "Webhook secret"
// @Param request body dto.WebhookRequest true "Webhook payload"
// @Success 200 {object} response.Message
// @Failure 401 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /api/payments/webhook [post]
func (handler *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".payment.Webhook")
	defer scope.End()

	if r.Header.Get(webhookSecretHeader) != handler.cfg.Payment.WebhookSecret {
		err := failure.Unauthorized("invalid webhook secret")
		scope.TraceError(err)
		response.WithError(w, err)

		return
	}

	req := dto.WebhookRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		response.WithError(w, err)

		return
	}

	payment, err := handler.settle(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to settle payment")
		response.WithError(w, err)

		return
	}

	if req.Status == dto.WebhookStatusSuccess {
		if err := handler.bookingService.Confirm(ctx, payment.BookingID); err != nil {
			scope.TraceError(err)
			log.Error().Err(err).Msg("failed to confirm booking")
			response.WithError(w, err)

			return
		}

		response.WithMessage(w, http.StatusOK, "payment confirmed, booking is now CONFIRMED")

		return
	}

	if err := handler.bookingService.Release(ctx, payment.BookingID); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to release booking")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "payment failed, booking released")
}

type settledPayment struct {
	BookingID string
}

func (handler *Handler) settle(ctx context.Context, req dto.WebhookRequest) (settledPayment, error) {
	if req.Status == dto.WebhookStatusSuccess {
		p, err := handler.paymentService.MarkSuccess(ctx, req.Reference)

		return settledPayment{BookingID: p.BookingID}, err
	}

	p, err := handler.paymentService.MarkFailed(ctx, req.Reference)

	return settledPayment{BookingID: p.BookingID}, err
}
