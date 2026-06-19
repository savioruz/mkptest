package http

import (
	"fmt"
	"net"
	"net/http"
	"oil/config"
	"oil/docs"
	"oil/infras/postgres"
	"oil/shared/constant"
	"oil/shared/logger"
	httpMiddleware "oil/transport/http/middleware"
	"oil/transport/http/response"
	"oil/transport/http/router"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/rs/zerolog/log"

	httpSwagger "github.com/swaggo/http-swagger"
)

type ServerState int

const (
	ServerStateReady ServerState = iota + 1
	ServerStateInGracePeriod
	ServerStateInCleanupPeriod
)

const (
	RouteHealthCheck = "/health"
	RouteSwaggerDocs = "/swagger/*"
)

type HTTP struct {
	Config         *config.Config
	Router         router.Router
	State          ServerState
	mux            *chi.Mux
	DB             *postgres.Connection
	appMiddleware  httpMiddleware.AppMiddleware
	authMiddleware httpMiddleware.AuthRole
}

func New(cfg *config.Config, r router.Router, db *postgres.Connection, appMiddleware httpMiddleware.AppMiddleware, authMiddleware httpMiddleware.AuthRole) *HTTP {
	return &HTTP{
		Config:         cfg,
		Router:         r,
		DB:             db,
		appMiddleware:  appMiddleware,
		authMiddleware: authMiddleware,
	}
}

func (h *HTTP) Serve() {
	h.setup()

	log.Info().Str("port", h.Config.Server.Port).Msg("Starting up HTTP server.")

	if err := http.ListenAndServe(net.JoinHostPort("0.0.0.0", h.Config.Server.Port), h.mux); err != nil {
		logger.ErrorWithStack(err)
	}
}

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.setup()
	h.mux.ServeHTTP(w, r)
}

func (h *HTTP) setup() {
	h.setupChi()
	h.setupMiddlewares()
	h.setupRoutes()
	h.setupSwaggerDocs()
	h.setupGracefulShutdown()
	h.State = ServerStateReady
}

func (h *HTTP) setupChi() {
	h.mux = chi.NewRouter()
}

func (h *HTTP) setupRoutes() {
	h.mux.Get(RouteHealthCheck, h.healthCheck)

	h.mux.
		With(
			h.authMiddleware.APIKey,
			h.authMiddleware.Auth,
			h.authMiddleware.RBAC,
		).
		Group(func(rc chi.Router) {
			h.Router.SetupRoutes(rc)
		})
}

func (h *HTTP) setupMiddlewares() {
	h.setupLogger()
	h.setupCORS()
	h.setupServerState()
	h.setupIdentity()
	h.setupRecover()
	h.setupRateLimit()
	h.setupTracing()

	h.logCORSConfigInfo()
}

func (h *HTTP) setupIdentity() {
	h.mux.Use(middleware.RequestID)
	h.mux.Use(middleware.RealIP)
}

func (h *HTTP) setupRecover() {
	h.mux.Use(middleware.Recoverer)
}

func (h *HTTP) setupServerState() {
	h.mux.Use(h.serverStateMiddleware)
}

func (h *HTTP) setupLogger() {
	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		h.mux.Use(middleware.Logger)
	} else {
		h.mux.Use(h.customJSONLogger())
	}
}

func (h *HTTP) customJSONLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a wrapped response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				// Log the request in JSON format
				log.Info().
					Str("method", r.Method).
					Str("url", r.URL.RequestURI()).
					Str("proto", r.Proto).
					Str("remote_addr", r.RemoteAddr).
					Str("user_agent", r.UserAgent()).
					Int("status", ww.Status()).
					Int("bytes", ww.BytesWritten()).
					Dur("duration", time.Since(start)).
					Str("request_id", middleware.GetReqID(r.Context())).
					Msg("HTTP Request")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func (h *HTTP) setupTracing() {
	h.mux.Use(h.appMiddleware.Tracing)
}

func (h *HTTP) setupRateLimit() {
	rateLimitConfig := h.Config.App.RateLimiter

	if rateLimitConfig.Enable {
		rateLimitMiddleware := h.appMiddleware.RateLimit()
		h.mux.Use(rateLimitMiddleware)

		log.Info().
			Bool("enabled", rateLimitConfig.Enable).
			Int("max_requests", rateLimitConfig.MaxRequests).
			Int("window_seconds", rateLimitConfig.WindowSeconds).
			Str("storage", "cache-redis").
			Msg("Rate limiting enabled with Redis cache storage")
	} else {
		log.Info().Msg("Rate limiting disabled")
	}
}

func (h *HTTP) setupSwaggerDocs() {
	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		docs.SwaggerInfo.Title = h.Config.App.Name
		h.mux.Get(RouteSwaggerDocs, httpSwagger.Handler())

		log.Info().Str("url", fmt.Sprintf("http://localhost:%s/swagger/index.html", h.Config.Server.Port)).Msg("Swagger docs available at")

		return
	}
}

func (h *HTTP) setupGracefulShutdown() {
	serverStateCh := make(chan os.Signal, 1)

	signal.Notify(serverStateCh, os.Interrupt, syscall.SIGTERM)

	go h.respondToSigterm(serverStateCh)
}

func (h *HTTP) respondToSigterm(done chan os.Signal) {
	<-done

	defer os.Exit(0)

	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		log.Warn().Msg("Received SIGTERM. Shutting down now.")

		return
	}

	shutdownConfig := h.Config.Server.Shutdown

	log.Info().Msg("Received SIGTERM.")
	log.Info().Int64("seconds", shutdownConfig.GracePeriodSeconds).Msg("Entering grace period.")

	h.State = ServerStateInGracePeriod

	time.Sleep(time.Duration(shutdownConfig.GracePeriodSeconds) * time.Second)

	log.Info().Int64("seconds", shutdownConfig.CleanupPeriodSeconds).Msg("Entering cleanup period.")

	h.State = ServerStateInCleanupPeriod

	time.Sleep(time.Duration(shutdownConfig.CleanupPeriodSeconds) * time.Second)

	log.Info().Msg("Cleaning up completed. Shutting down now.")
}

func (h *HTTP) serverStateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch h.State {
		case ServerStateReady:
			// Server is ready to serve, don't do anything.
			next.ServeHTTP(writer, request)
		case ServerStateInGracePeriod:
			// Server is in grace period. Issue a warning message and continue
			// serving as usual.
			log.Warn().Msg("SERVER IS IN GRACE PERIOD")
			next.ServeHTTP(writer, request)
		case ServerStateInCleanupPeriod:
			// Server is in cleanup period. Stop the request from actually
			// invoking any domain services and respond appropriately.
			response.WithPreparingShutdown(writer)
		}
	})
}

func (h *HTTP) setupCORS() {
	corsConfig := h.Config.App.CORS
	if corsConfig.Enable {
		h.mux.Use(cors.Handler(cors.Options{
			AllowedOrigins:   corsConfig.AllowedOrigins,
			AllowedMethods:   corsConfig.AllowedMethods,
			AllowedHeaders:   corsConfig.AllowedHeaders,
			AllowCredentials: corsConfig.AllowCredentials,
			MaxAge:           corsConfig.MaxAgeSeconds,
		}))
	}
}

func (h *HTTP) logCORSConfigInfo() {
	corsConfig := h.Config.App.CORS
	corsHeaderInfo := "CORS Header"

	if corsConfig.Enable {
		log.Info().Msg("CORS Headers and Handlers are enabled.")
		log.Info().Str(corsHeaderInfo, fmt.Sprintf("Access-Control-Allow-Credentials: %t", corsConfig.AllowCredentials)).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Headers: "+strings.Join(corsConfig.AllowedHeaders, ", ")).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Methods: "+strings.Join(corsConfig.AllowedMethods, ", ")).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Origin: "+strings.Join(corsConfig.AllowedOrigins, ", ")).Msg("")
		log.Info().Str(corsHeaderInfo, fmt.Sprintf("Access-Control-Max-Age: %d", corsConfig.MaxAgeSeconds)).Msg("")
	} else {
		log.Info().Msg("CORS Headers are disabled.")
	}
}

// HealthCheck performs a health check on the server.
// @Summary Health Check
// @Description Health Check APIEndpoint
// @Tags service
// @Produce json
// @Accept json
// @Success 200 {object} response.Message
// @Router /health [get]
func (h *HTTP) healthCheck(writer http.ResponseWriter, _ *http.Request) {
	if err := h.DB.Read.Ping(); err != nil {
		logger.ErrorWithStack(err)
		response.WithUnhealthy(writer)

		return
	}

	response.WithMessage(writer, http.StatusOK, "ok")
}
