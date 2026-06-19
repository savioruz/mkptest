// Package seatlock provides short-lived, all-or-nothing seat holds in Redis.
// A hold is the first line of defence against double-booking (fast, UX-facing);
// the booking_seats partial unique index is the second (correct, DB-enforced).
package seatlock

//go:generate go run go.uber.org/mock/mockgen -source=./seatlock.go -destination=./mocks/seatlock_mock.go -package=mocks

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "seat"

// holdScript holds every seat only if none of them is already held. The two
// loops run inside one atomic Lua execution, so concurrent callers can never
// both succeed on an overlapping seat set.
var holdScript = redis.NewScript(`
for i = 1, #KEYS do
  if redis.call('EXISTS', KEYS[i]) == 1 then
    return 0
  end
end
for i = 1, #KEYS do
  redis.call('SET', KEYS[i], ARGV[1], 'EX', ARGV[2])
end
return 1
`)

// Locker manages seat holds for a schedule.
type Locker interface {
	// Hold atomically holds all seatIDs for ttlSeconds. Returns false if any
	// seat is already held (no seat is held in that case).
	Hold(ctx context.Context, scheduleID, userID string, seatIDs []string, ttlSeconds int) (bool, error)
	// Release removes the holds for the given seats (no-op if absent).
	Release(ctx context.Context, scheduleID string, seatIDs []string) error
	// HeldSeatIDs returns, for the given candidate seats, which are currently held.
	HeldSeatIDs(ctx context.Context, scheduleID string, seatIDs []string) (map[string]bool, error)
}

type lockerImpl struct {
	client *redis.Client
}

func New(client *redis.Client) Locker {
	return &lockerImpl{client: client}
}

func seatKey(scheduleID, seatID string) string {
	return fmt.Sprintf("%s:%s:%s", keyPrefix, scheduleID, seatID)
}

func (l *lockerImpl) Hold(ctx context.Context, scheduleID, userID string, seatIDs []string, ttlSeconds int) (bool, error) {
	if len(seatIDs) == 0 {
		return false, nil
	}

	keys := make([]string, len(seatIDs))
	for i, s := range seatIDs {
		keys[i] = seatKey(scheduleID, s)
	}

	res, err := holdScript.Run(ctx, l.client, keys, userID, ttlSeconds).Int()
	if err != nil {
		return false, fmt.Errorf("failed to run seat hold script: %w", err)
	}

	return res == 1, nil
}

func (l *lockerImpl) Release(ctx context.Context, scheduleID string, seatIDs []string) error {
	if len(seatIDs) == 0 {
		return nil
	}

	keys := make([]string, len(seatIDs))
	for i, s := range seatIDs {
		keys[i] = seatKey(scheduleID, s)
	}

	if err := l.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to release seat holds: %w", err)
	}

	return nil
}

func (l *lockerImpl) HeldSeatIDs(ctx context.Context, scheduleID string, seatIDs []string) (map[string]bool, error) {
	held := make(map[string]bool, len(seatIDs))
	if len(seatIDs) == 0 {
		return held, nil
	}

	keys := make([]string, len(seatIDs))
	for i, s := range seatIDs {
		keys[i] = seatKey(scheduleID, s)
	}

	vals, err := l.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read seat holds: %w", err)
	}

	for i, v := range vals {
		if v != nil {
			held[seatIDs[i]] = true
		}
	}

	return held, nil
}
