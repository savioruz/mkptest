package events

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	notificationConsumer "oil/events/notification/service"
	refundConsumer "oil/events/refund/service"

	"github.com/rs/zerolog/log"
)

// Consumers aggregates all domain Kafka consumers. They are wired via DI and
// launched from cmd/app/main.go alongside the HTTP server (single process).
type Consumers struct {
	Refund       refundConsumer.Consumer
	Notification notificationConsumer.Consumer
}

// New constructs the aggregate Consumers.
func New(refund refundConsumer.Consumer, notification notificationConsumer.Consumer) *Consumers {
	return &Consumers{
		Refund:       refund,
		Notification: notification,
	}
}

// Start starts all domain event consumers and wires graceful shutdown.
func (c *Consumers) Start() {
	log.Info().Msg("Starting consumers...")

	kafkaConsumerState := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(kafkaConsumerState, os.Interrupt, syscall.SIGTERM)

	c.Refund.Start(ctx)
	c.Notification.Start(ctx)

	go func() {
		<-kafkaConsumerState
		log.Info().Msg("Received SIGTERM. Shutting down now.")
		cancel()
	}()
}
