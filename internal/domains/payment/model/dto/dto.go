package dto

const (
	WebhookStatusSuccess = "success"
	WebhookStatusFailed  = "failed"
)

// WebhookRequest is the payload the (mock) gateway posts to settle a charge.
type WebhookRequest struct {
	Reference string `json:"reference" validate:"required"`
	Status    string `json:"status"    validate:"required,oneof=success failed"`
}
