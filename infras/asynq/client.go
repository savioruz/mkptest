// Package asynq provides a wrapper around github.com/hibiken/asynq for task queue operations.
package asynq

import (
	"context"
	"fmt"
	"time"

	"oil/config"

	"github.com/hibiken/asynq"
)

// RedisConnOpt configures the Redis connection used by asynq.
type RedisConnOpt struct {
	Addr     string
	Password string
	DB       int
}

func (r RedisConnOpt) toAsynq() asynq.RedisConnOpt {
	return asynq.RedisClientOpt{
		Addr:     r.Addr,
		Password: r.Password,
		DB:       r.DB,
	}
}

// TaskInfo holds metadata about an enqueued task.
type TaskInfo struct {
	ID string
}

// Client defines the interface for enqueueing tasks.
type Client interface {
	Enqueue(ctx context.Context, taskType string, payload []byte, processAt time.Time) (*TaskInfo, error)
}

type clientImpl struct {
	client *asynq.Client
}

// NewClient creates a new asynq client wrapper.
func NewClient(cfg *config.Config) Client {
	opt := RedisConnOpt{
		Addr:     cfg.Cache.Redis.Primary.Host + ":" + cfg.Cache.Redis.Primary.Port,
		Password: cfg.Cache.Redis.Primary.Password,
		DB:       cfg.Cache.Redis.Primary.DB,
	}

	return &clientImpl{
		client: asynq.NewClient(opt.toAsynq()),
	}
}

func (c *clientImpl) Enqueue(ctx context.Context, taskType string, payload []byte, processAt time.Time) (*TaskInfo, error) {
	task := asynq.NewTask(taskType, payload, asynq.ProcessAt(processAt))

	info, err := c.client.EnqueueContext(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return &TaskInfo{ID: info.ID}, nil
}
