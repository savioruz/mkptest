//go:build wireinject
// +build wireinject

package di

import (
	"oil/config"
	"oil/events"
	"oil/infras/asynq"
	"oil/infras/jwt"
	"oil/infras/kafka"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/infras/redis"
	"oil/infras/s3"
	"oil/permissions"
	"oil/shared/cache"
	"oil/transport/http"
	"oil/transport/http/middleware"
	"oil/transport/http/router"

	"github.com/google/wire"

	authService "oil/internal/domains/auth/service"
	userRepository "oil/internal/domains/user/repository"
	authHandler "oil/internal/handlers/auth"

	movieRepository "oil/internal/domains/movie/repository"
	movieService "oil/internal/domains/movie/service"
	movieHandler "oil/internal/handlers/movie"

	cinemaRepository "oil/internal/domains/cinema/repository"
	cinemaService "oil/internal/domains/cinema/service"
	cinemaHandler "oil/internal/handlers/cinema"

	seatRepository "oil/internal/domains/seat/repository"
	studioRepository "oil/internal/domains/studio/repository"
	studioService "oil/internal/domains/studio/service"
	studioHandler "oil/internal/handlers/studio"

	scheduleRepository "oil/internal/domains/schedule/repository"
	scheduleService "oil/internal/domains/schedule/service"
	scheduleHandler "oil/internal/handlers/schedule"

	paymentGateway "oil/infras/payment"
	paymentRepository "oil/internal/domains/payment/repository"
	paymentService "oil/internal/domains/payment/service"
	paymentHandler "oil/internal/handlers/payment"

	bookingRepository "oil/internal/domains/booking/repository"
	bookingService "oil/internal/domains/booking/service"
	bookingHandler "oil/internal/handlers/booking"

	refundRepository "oil/internal/domains/refund/repository"
	refundService "oil/internal/domains/refund/service"
	refundHandler "oil/internal/handlers/refund"

	notificationConsumer "oil/events/notification/service"
	refundConsumer "oil/events/refund/service"

	"oil/internal/workers"
	"oil/shared/seatlock"
)

var configurations = wire.NewSet(
	config.Get,
	permissions.Get,
)

var infrastructures = wire.NewSet(
	postgres.New,
	otel.New,
	redis.New,
	s3.New,
	jwt.New,
	kafka.New,
	asynq.NewClient,
	asynq.NewServer,
)

var middlewares = wire.NewSet(
	middleware.NewAppMiddleware,
	middleware.NewAuthRoleMiddleware,
)

var sharedHelpers = wire.NewSet(
	cache.NewRedisCache,
	seatlock.New,
)

var authDomain = wire.NewSet(
	userRepository.New,
	authService.New,
)

var movieDomain = wire.NewSet(
	movieRepository.New,
	movieService.New,
)

var cinemaDomain = wire.NewSet(
	cinemaRepository.New,
	cinemaService.New,
)

var studioDomain = wire.NewSet(
	seatRepository.New,
	studioRepository.New,
	studioService.New,
)

var scheduleDomain = wire.NewSet(
	scheduleRepository.New,
	scheduleService.New,
)

var paymentDomain = wire.NewSet(
	paymentGateway.NewMockGateway,
	paymentRepository.New,
	paymentService.New,
)

var bookingDomain = wire.NewSet(
	bookingRepository.New,
	bookingRepository.NewSeat,
	bookingService.New,
)

var refundDomain = wire.NewSet(
	refundRepository.New,
	refundService.New,
)

var domains = wire.NewSet(
	authDomain,
	movieDomain,
	cinemaDomain,
	studioDomain,
	scheduleDomain,
	paymentDomain,
	bookingDomain,
	refundDomain,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	authHandler.New,
	movieHandler.New,
	cinemaHandler.New,
	studioHandler.New,
	scheduleHandler.New,
	bookingHandler.New,
	paymentHandler.New,
	refundHandler.New,
	router.New,
)

var eventConsumers = wire.NewSet(
	refundConsumer.New,
	notificationConsumer.New,
	events.New,
)

func InitializeService() *http.HTTP {
	wire.Build(
		configurations,
		infrastructures,
		middlewares,
		sharedHelpers,
		domains,
		routing,
		http.New,
	)

	return &http.HTTP{}
}

// InitializeEvents builds the aggregate Kafka consumers, started from cmd/app.
// Provider sets (config/infra/messaging) are added here as consumers gain deps.
func InitializeEvents() *events.Consumers {
	wire.Build(
		configurations,
		infrastructures,
		paymentDomain,
		bookingDomain,
		refundDomain,
		eventConsumers,
	)

	return &events.Consumers{}
}

// InitializeWorkers builds the Asynq worker registry, started from cmd/app. It
// shares Redis with the enqueuing services, so its own service instances are fine.
func InitializeWorkers() *workers.Registry {
	wire.Build(
		configurations,
		infrastructures,
		sharedHelpers,
		scheduleDomain,
		studioDomain,
		paymentDomain,
		bookingDomain,
		workers.New,
	)

	return &workers.Registry{}
}
