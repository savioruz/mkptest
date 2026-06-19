// Package workers registers and runs the Asynq task handlers. It runs in-process
// with the HTTP server (started from cmd/app) and consumes tasks enqueued by the
// domain services via Redis.
package workers

import (
	"context"
	"encoding/json"

	"oil/infras/asynq"
	bookingSvc "oil/internal/domains/booking/service"
	"oil/shared/constant"

	"github.com/rs/zerolog/log"
)

type bookingPayload struct {
	BookingID string `json:"booking_id"`
}

// Registry wires task types to their handlers and runs the Asynq server.
type Registry struct {
	server  asynq.Server
	booking bookingSvc.Booking
}

func New(server asynq.Server, booking bookingSvc.Booking) *Registry {
	return &Registry{server: server, booking: booking}
}

// Start registers all handlers and runs the Asynq server (non-blocking).
func (r *Registry) Start() {
	r.server.RegisterHandler(constant.TaskReleaseHold, r.handleReleaseHold)
	r.server.RegisterHandler(constant.TaskSendETicket, r.handleSendETicket)
	r.server.RegisterHandler(constant.TaskNotifyRefund, r.handleNotifyRefund)

	log.Info().Msg("Starting Asynq workers...")
	r.server.Start(context.Background())
}

func (r *Registry) Stop() {
	r.server.Stop()
}

// handleReleaseHold expires a still-pending booking once its hold window passes.
func (r *Registry) handleReleaseHold(ctx context.Context, _ string, payload []byte) error {
	var p bookingPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Error().Err(err).Msg("release_hold: bad payload")

		return err
	}

	if err := r.booking.Release(ctx, p.BookingID); err != nil {
		log.Error().Err(err).Str("booking_id", p.BookingID).Msg("release_hold failed")

		return err
	}

	return nil
}

// handleSendETicket is a stub for e-ticket/QR generation + delivery.
func (r *Registry) handleSendETicket(_ context.Context, _ string, payload []byte) error {
	var p bookingPayload
	_ = json.Unmarshal(payload, &p)

	log.Info().Str("booking_id", p.BookingID).Msg("e-ticket generated and sent (stub)")

	return nil
}

// handleNotifyRefund is a stub for refund-completed customer notification.
func (r *Registry) handleNotifyRefund(_ context.Context, _ string, payload []byte) error {
	log.Info().RawJSON("payload", payload).Msg("refund notification sent (stub)")

	return nil
}
