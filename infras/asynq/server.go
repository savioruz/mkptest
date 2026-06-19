package asynq

import (
	"context"
	"time"

	"oil/config"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const (
	defaultConcurrency   = 10
	retryDelayMultiplier = 10
	defaultQueueWeight   = 1
)

// HandlerFunc is the signature for task handlers.
type HandlerFunc func(ctx context.Context, taskType string, payload []byte) error

// Server defines the interface for running task workers.
type Server interface {
	RegisterHandler(taskType string, handler HandlerFunc)
	Start(ctx context.Context)
	Stop()
}

type serverImpl struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

// NewServer creates a new asynq server wrapper.
func NewServer(cfg *config.Config) Server {
	opt := RedisConnOpt{
		Addr:     cfg.Cache.Redis.Primary.Host + ":" + cfg.Cache.Redis.Primary.Port,
		Password: cfg.Cache.Redis.Primary.Password,
		DB:       cfg.Cache.Redis.Primary.DB,
	}

	srv := asynq.NewServer(
		opt.toAsynq(),
		asynq.Config{
			Concurrency: defaultConcurrency,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(n*retryDelayMultiplier) * time.Second
			},
			Queues: map[string]int{
				"default": defaultQueueWeight,
			},
		},
	)

	return &serverImpl{
		srv: srv,
		mux: asynq.NewServeMux(),
	}
}

func (s *serverImpl) RegisterHandler(taskType string, handler HandlerFunc) {
	s.mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		return handler(ctx, task.Type(), task.Payload())
	})
}

func (s *serverImpl) Start(_ context.Context) {
	go func() {
		if err := s.srv.Run(s.mux); err != nil {
			log.Error().Err(err).Msg("asynq server failed")
		}
	}()
}

func (s *serverImpl) Stop() {
	s.srv.Shutdown()
}
