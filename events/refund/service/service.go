// Package service hosts the refund-related Kafka consumers: it turns
// schedule.cancelled into per-booking refund requests, and processes each
// refund.requested event. Handlers are wrapped with a bounded retry.
package service

import (
	"context"
	"encoding/json"
	"time"

	"oil/config"
	"oil/infras/kafka"
	"oil/infras/otel"
	refundSvc "oil/internal/domains/refund/service"
	"oil/shared/constant"
	"oil/shared/wrappers"

	kafkaGo "github.com/segmentio/kafka-go"
)

type Consumer interface {
	Start(ctx context.Context)
}

type consumerImpl struct {
	cfg       *config.Config
	otel      otel.Otel
	kafka     kafka.Client
	refundSvc refundSvc.Refund
}

func New(cfg *config.Config, otel otel.Otel, kafka kafka.Client, refundSvc refundSvc.Refund) Consumer {
	return &consumerImpl{cfg: cfg, otel: otel, kafka: kafka, refundSvc: refundSvc}
}

func (s *consumerImpl) settings() config.ConsumerConfig {
	c := s.cfg.Kafka.Refund
	if c.ConsumerGroup == "" {
		c.ConsumerGroup = s.cfg.Kafka.ConsumerGroup + "-refund"
	}

	return c
}

func (s *consumerImpl) Start(ctx context.Context) {
	if !s.cfg.Kafka.Enable {
		return
	}

	group := s.settings().ConsumerGroup

	go s.kafka.Consume(ctx, group, constant.TopicScheduleCancelled, s.handleScheduleCancelled(ctx))
	go s.kafka.Consume(ctx, group, constant.TopicRefundRequested, s.handleRefundRequested(ctx))
}

type scheduleCancelledPayload struct {
	ScheduleID string `json:"schedule_id"`
}

func (s *consumerImpl) handleScheduleCancelled(ctx context.Context) func(kafkaGo.Message) {
	return func(message kafkaGo.Message) {
		ctx, scope := s.otel.NewScope(ctx, constant.OtelEventScopeName, constant.OtelEventScopeName+".schedule_cancelled")
		defer scope.End()

		var p scheduleCancelledPayload
		if err := json.Unmarshal(message.Value, &p); err != nil {
			scope.TraceError(err)

			return
		}

		cfg := s.settings()
		if _, err := wrappers.Retry(cfg.MaxRetry, backoff(cfg), func() (any, error) {
			return nil, s.refundSvc.CreateForCancelledSchedule(ctx, p.ScheduleID)
		}); err != nil {
			scope.TraceError(err)
		}
	}
}

type refundRequestedPayload struct {
	RefundID string `json:"refund_id"`
}

func (s *consumerImpl) handleRefundRequested(ctx context.Context) func(kafkaGo.Message) {
	return func(message kafkaGo.Message) {
		ctx, scope := s.otel.NewScope(ctx, constant.OtelEventScopeName, constant.OtelEventScopeName+".refund_requested")
		defer scope.End()

		var p refundRequestedPayload
		if err := json.Unmarshal(message.Value, &p); err != nil {
			scope.TraceError(err)

			return
		}

		cfg := s.settings()
		if _, err := wrappers.Retry(cfg.MaxRetry, backoff(cfg), func() (any, error) {
			return nil, s.refundSvc.Process(ctx, p.RefundID)
		}); err != nil {
			scope.TraceError(err)
		}
	}
}

func backoff(c config.ConsumerConfig) time.Duration {
	return time.Duration(c.BackoffDuration) * time.Second
}
