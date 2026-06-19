// Package service hosts notification/analytics consumers. Each reader's
// consumer group and topic come from config; the log label is derived from the
// message topic (never hard-coded). Handlers are retry-wrapped.
package service

import (
	"context"
	"time"

	"oil/config"
	"oil/infras/kafka"
	"oil/infras/otel"
	"oil/shared/constant"
	"oil/shared/wrappers"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/rs/zerolog/log"
)

type Consumer interface {
	Start(ctx context.Context)
}

type consumerImpl struct {
	cfg   *config.Config
	otel  otel.Otel
	kafka kafka.Client
}

func New(cfg *config.Config, otel otel.Otel, kafka kafka.Client) Consumer {
	return &consumerImpl{cfg: cfg, otel: otel, kafka: kafka}
}

func (s *consumerImpl) Start(ctx context.Context) {
	if !s.cfg.Kafka.Enable {
		return
	}

	go s.kafka.Consume(ctx, s.cfg.Kafka.Notification.ConsumerGroup, s.cfg.Kafka.Topics.Notification, s.handle(ctx, s.cfg.Kafka.Notification))
	go s.kafka.Consume(ctx, s.cfg.Kafka.Refund.ConsumerGroup, s.cfg.Kafka.Topics.Refund, s.handle(ctx, s.cfg.Kafka.Refund))
}

func (s *consumerImpl) handle(ctx context.Context, cc config.ConsumerConfig) func(kafkaGo.Message) {
	return func(message kafkaGo.Message) {
		_, scope := s.otel.NewScope(ctx, constant.OtelEventScopeName, constant.OtelEventScopeName+"."+message.Topic)
		defer scope.End()

		if _, err := wrappers.Retry(cc.MaxRetry, time.Duration(cc.BackoffDuration)*time.Second, func() (any, error) {
			// Label is the topic itself — not hard-coded.
			log.Info().Str("topic", message.Topic).RawJSON("event", message.Value).Msg(message.Topic)

			return nil, nil
		}); err != nil {
			scope.TraceError(err)
		}
	}
}
