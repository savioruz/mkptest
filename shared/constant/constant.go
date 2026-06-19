package constant

import (
	"time"
)

const (
	ContextGuest = "guest"
)

// Context key types to avoid collisions
type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
	ContextKeyTokenID   contextKey = "token_id"
)

const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

const (
	RequestParamPage    = "page"
	RequestParamLimit   = "limit"
	RequestParamSortBy  = "sort_by"
	RequestParamSortDir = "sort_dir"
)

const (
	RequestParamID   = "id"
	RequestParamCode = "code"

	RequestMaxMemory = 10 << 20 // 10 MB
)

const (
	DefaultValuePage    = 1
	DefaultValueLimit   = 10
	DefaultValueSortBy  = "created_at"
	DefaultValueSortDir = "DESC"
)

const (
	FieldActive     = "active"
	FieldCreatedAt  = "created_at"
	FieldCreatedBy  = "created_by"
	FieldModifiedAt = "modified_at"
	FieldModifiedBy = "modified_by"
)

const (
	PqErrorCodeUniqueViolation    = "23505"
	PqErrorCodeFkViolation        = "23503"
	PqErrorCodeExclusionViolation = "23P01"
	PqErrorCodeCheckViolation     = "23514"
)

const (
	FkUsed = "another entity"
)

// Kafka topics — durable, replayable facts about things that happened.
const (
	TopicTicketSold        = "ticket.sold"
	TopicScheduleCancelled = "schedule.cancelled"
	TopicRefundRequested   = "refund.requested"
	TopicSeatRestocked     = "seat.restocked"
)

// Asynq task types — commands to run once, retryable, possibly deferred.
const (
	TaskReleaseHold  = "booking.release_hold"
	TaskSendETicket  = "booking.send_eticket"
	TaskNotifyRefund = "notify.refund"
)

const (
	DateFormat = time.RFC3339
)

const (
	MinutesToSeconds = 60
)

const (
	OtelServiceScopeName    = "service"
	OtelRepositoryScopeName = "repository"
	OtelHandlerScopeName    = "handler"
	OtelEventScopeName      = "event"
	OtelExternalScopeName   = "external"

	OtelQueryAttributeKey = "query"
	OtelS3ScopeName       = "s3"

	OtelUserScopeName = "user"
)

const (
	RequestHeaderAuthorization      = "Authorization"
	RequestHeaderUserAgent          = "User-Agent"
	RequestHeaderContentType        = "Content-Type"
	RequestHeaderRateLimit          = "X-RateLimit-Limit"
	RequestHeaderRateLimitRemaining = "X-RateLimit-Remaining"
	RequestHeaderRateLimitWindow    = "X-RateLimit-Window"
	RequestHeaderRequestID          = "X-Request-ID"
	RequestHeaderForwardedFor       = "X-Forwarded-For"
	RequestHeaderRealIP             = "X-Real-IP"
	RequestHeaderAPIKey             = "X-API-Key"
)

const (
	ContentTypeJSON              = "application/json"
	ContentTypeFormURLEncoded    = "application/x-www-form-urlencoded"
	ContentTypeMultipartFormData = "multipart/form-data"
	FormFile                     = "file"
)

const (
	ResponseErrorPrepareShutdown      = "SERVER PREPARING TO SHUT DOWN"
	ResponseErrorUnhealthy            = "SERVER UNHEALTHY"
	ResponseErrorRequestLimitExceeded = "REQUEST LIMIT EXCEEDED"
)

const (
	ServerEnvDevelopment = "development"
	ServerEnvProduction  = "production"
)

const (
	Asterix = "*"
	Empty   = ""
)
