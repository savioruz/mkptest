package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	otelMocks "oil/infras/otel/mocks"
	gatewayMocks "oil/infras/payment/mocks"
	"oil/internal/domains/payment/model"
	paymentMocks "oil/internal/domains/payment/mocks"
	"oil/internal/domains/payment/service"
)

func newService(t *testing.T) (service.Payment, *paymentMocks.MockPayment, *gatewayMocks.MockGateway) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := paymentMocks.NewMockPayment(ctrl)
	gw := gatewayMocks.NewMockGateway(ctrl)

	return service.New(repo, gw, otelMocks.NewOtel()), repo, gw
}

func TestPaymentService_CreateCharge_IdempotentReturnsExisting(t *testing.T) {
	svc, repo, gw := newService(t)

	existing := model.Payment{ID: "pay-1", Reference: "MOCK-AAA", IdempotencyKey: "key-1", Status: model.StatusPending}

	// An existing payment with the same idempotency key is returned as-is.
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(existing, nil)
	// Gateway must NOT be charged again, and no new row inserted.
	gw.EXPECT().Charge(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Times(0)

	res, err := svc.CreateCharge(context.Background(), "booking-1", 50000, "key-1")

	assert.NoError(t, err)
	assert.Equal(t, "pay-1", res.ID)
	assert.Equal(t, "MOCK-AAA", res.Reference)
}

func TestPaymentService_CreateCharge_NewChargesGateway(t *testing.T) {
	svc, repo, gw := newService(t)

	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(model.Payment{}, nil)
	gw.EXPECT().Charge(gomock.Any(), "booking-1", 50000.0).Return("MOCK-NEW", nil)
	repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	res, err := svc.CreateCharge(context.Background(), "booking-1", 50000, "key-2")

	assert.NoError(t, err)
	assert.Equal(t, "MOCK-NEW", res.Reference)
	assert.Equal(t, model.StatusPending, res.Status)
}

func TestPaymentService_MarkSuccess_IdempotentOnSettled(t *testing.T) {
	svc, repo, _ := newService(t)

	// Already-success payment: repeated webhook is a no-op (no Update).
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(model.Payment{ID: "pay-1", Reference: "MOCK-AAA", Status: model.StatusSuccess}, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	res, err := svc.MarkSuccess(context.Background(), "MOCK-AAA")

	assert.NoError(t, err)
	assert.Equal(t, model.StatusSuccess, res.Status)
}

func TestPaymentService_MarkSuccess_TransitionsPending(t *testing.T) {
	svc, repo, _ := newService(t)

	repo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(model.Payment{ID: "pay-1", Reference: "MOCK-AAA", Status: model.StatusPending}, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	res, err := svc.MarkSuccess(context.Background(), "MOCK-AAA")

	assert.NoError(t, err)
	assert.Equal(t, model.StatusSuccess, res.Status)
	assert.NotNil(t, res.PaidAt)
}
