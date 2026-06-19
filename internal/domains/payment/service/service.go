package service

//go:generate go run go.uber.org/mock/mockgen -source=./service.go -destination=./mocks/service_mock.go -package=mocks

import (
	"context"
	"fmt"

	gateway "oil/infras/payment"
	"oil/infras/otel"
	"oil/internal/domains/payment/model"
	paymentRepo "oil/internal/domains/payment/repository"
	"oil/shared"
	"oil/shared/constant"
	"oil/shared/failure"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const systemActor = "system"

type Payment interface {
	// CreateCharge initiates (or returns the existing, by idempotency key) charge for a booking.
	CreateCharge(ctx context.Context, bookingID string, amount float64, idempotencyKey string) (model.Payment, error)
	GetByReference(ctx context.Context, reference string) (model.Payment, error)
	MarkSuccess(ctx context.Context, reference string) (model.Payment, error)
	MarkFailed(ctx context.Context, reference string) (model.Payment, error)
}

type serviceImpl struct {
	paymentRepo paymentRepo.Payment
	gateway     gateway.Gateway
	otel        otel.Otel
}

func New(paymentRepo paymentRepo.Payment, gateway gateway.Gateway, otel otel.Otel) Payment {
	return &serviceImpl{
		paymentRepo: paymentRepo,
		gateway:     gateway,
		otel:        otel,
	}
}

func (s *serviceImpl) CreateCharge(ctx context.Context, bookingID string, amount float64, idempotencyKey string) (res model.Payment, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".payment.CreateCharge")
	defer scope.End()
	defer scope.TraceIfError(err)

	// Idempotency: a repeated charge with the same key returns the first payment.
	existing, err := s.paymentRepo.Get(ctx, shared.FilterByID(idempotencyKey, model.FieldIdempotencyKey, model.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to check existing payment: %w", err)
	}

	if existing.ID != "" {
		return existing, nil
	}

	reference, err := s.gateway.Charge(ctx, bookingID, amount)
	if err != nil {
		return res, fmt.Errorf("failed to charge: %w", err)
	}

	now := timezone.Now()
	payment := model.Payment{
		ID:             uuid.NewString(),
		BookingID:      bookingID,
		Amount:         amount,
		Status:         model.StatusPending,
		Reference:      reference,
		IdempotencyKey: idempotencyKey,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  systemActor,
			ModifiedBy: systemActor,
		},
	}

	if err = s.paymentRepo.Insert(ctx, payment); err != nil {
		return res, fmt.Errorf("failed to insert payment: %w", err)
	}

	return payment, nil
}

func (s *serviceImpl) GetByReference(ctx context.Context, reference string) (res model.Payment, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".payment.GetByReference")
	defer scope.End()
	defer scope.TraceIfError(err)

	payment, err := s.paymentRepo.Get(ctx, shared.FilterByID(reference, model.FieldReference, model.TableName))
	if err != nil {
		return res, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.ID == "" {
		return res, failure.NotFound("payment not found")
	}

	return payment, nil
}

func (s *serviceImpl) MarkSuccess(ctx context.Context, reference string) (res model.Payment, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".payment.MarkSuccess")
	defer scope.End()
	defer scope.TraceIfError(err)

	return s.transition(ctx, reference, model.StatusSuccess)
}

func (s *serviceImpl) MarkFailed(ctx context.Context, reference string) (res model.Payment, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".payment.MarkFailed")
	defer scope.End()
	defer scope.TraceIfError(err)

	return s.transition(ctx, reference, model.StatusFailed)
}

func (s *serviceImpl) transition(ctx context.Context, reference, status string) (model.Payment, error) {
	payment, err := s.paymentRepo.Get(ctx, shared.FilterByID(reference, model.FieldReference, model.TableName))
	if err != nil {
		return payment, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.ID == "" {
		return payment, failure.NotFound("payment not found")
	}

	// Idempotent: once settled, repeated webhooks are no-ops.
	if payment.Status != model.StatusPending {
		return payment, nil
	}

	now := timezone.Now()
	fields := map[string]any{
		model.FieldStatus:        status,
		constant.FieldModifiedAt: now,
		constant.FieldModifiedBy: systemActor,
	}

	if status == model.StatusSuccess {
		fields[model.FieldPaidAt] = now
	}

	filter := shared.FilterByID(reference, model.FieldReference, model.TableName)
	if err = s.paymentRepo.Update(ctx, fields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update payment status")

		return payment, fmt.Errorf("failed to update payment: %w", err)
	}

	payment.Status = status
	if status == model.StatusSuccess {
		payment.PaidAt = &now
	}

	return payment, nil
}
