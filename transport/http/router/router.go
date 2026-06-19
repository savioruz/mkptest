package router

import (
	"oil/internal/handlers/auth"
	"oil/internal/handlers/booking"
	"oil/internal/handlers/cinema"
	"oil/internal/handlers/movie"
	"oil/internal/handlers/payment"
	"oil/internal/handlers/refund"
	"oil/internal/handlers/schedule"
	"oil/internal/handlers/studio"

	"github.com/go-chi/chi/v5"
)

type DomainHandlers struct {
	Auth     auth.Handler
	Movie    movie.Handler
	Cinema   cinema.Handler
	Studio   studio.Handler
	Schedule schedule.Handler
	Booking  booking.Handler
	Payment  payment.Handler
	Refund   refund.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/api", func(routerGroup chi.Router) {
		r.DomainHandlers.Auth.Router(routerGroup)
		r.DomainHandlers.Movie.Router(routerGroup)
		r.DomainHandlers.Cinema.Router(routerGroup)
		r.DomainHandlers.Studio.Router(routerGroup)
		r.DomainHandlers.Schedule.Router(routerGroup)
		r.DomainHandlers.Booking.Router(routerGroup)
		r.DomainHandlers.Payment.Router(routerGroup)
		r.DomainHandlers.Refund.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
