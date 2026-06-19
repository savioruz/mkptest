// Package payment provides a payment gateway abstraction. The default
// implementation is an internal mock: Charge returns a pending reference and
// settlement is driven by a webhook callback the reviewer POSTs manually, so
// the whole flow runs with no external account or credentials.
package payment

//go:generate go run go.uber.org/mock/mockgen -source=./gateway.go -destination=./mocks/gateway_mock.go -package=mocks

import (
	"context"
	"strings"

	"oil/config"

	"github.com/google/uuid"
)

// Gateway is the payment provider abstraction (mock or, in future, real).
type Gateway interface {
	// Charge initiates a charge and returns an opaque provider reference. The
	// charge starts as pending; settlement arrives later via webhook.
	Charge(ctx context.Context, bookingID string, amount float64) (reference string, err error)
	// Refund refunds a previously settled charge. Idempotency is enforced by
	// the caller via the refunds.idempotency_key unique constraint.
	Refund(ctx context.Context, reference string, amount float64) error
}

type mockGateway struct {
	cfg *config.Config
}

// NewMockGateway returns the internal mock gateway.
func NewMockGateway(cfg *config.Config) Gateway {
	return &mockGateway{cfg: cfg}
}

func (m *mockGateway) Charge(_ context.Context, _ string, _ float64) (string, error) {
	ref := "MOCK-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:16])

	return ref, nil
}

func (m *mockGateway) Refund(_ context.Context, _ string, _ float64) error {
	// The mock always settles refunds successfully.
	return nil
}
