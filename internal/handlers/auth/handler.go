package auth

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/auth/model/dto"
	"oil/internal/domains/auth/service"
	"oil/shared/constant"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Auth
	otel    otel.Otel
}

func New(service service.Auth, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.Post("/refresh-token", handler.RefreshToken)
	})
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user with the provided details.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register Request"
// @Success 201 {object} response.Message "User registered successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/auth/register [post]
func (handler *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Register")
	defer scope.End()

	req := dto.RegisterRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Register(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create todo")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User registered successfully")

	response.WithMessage(w, http.StatusCreated, "User registered successfully")
}

// Login handles user login
// @Summary Login a user
// @Description Login a user with the provided credentials.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login Request"
// @Success 200 {object} dto.LoginResponse "User logged in successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/auth/login [post]
func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Login")
	defer scope.End()

	req := dto.LoginRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	res, err := handler.service.Login(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to login user")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User logged in successfully")

	response.WithJSON(w, http.StatusOK, res)
}

// RefreshToken handles token refresh
// @Summary Refresh user token
// @Description Refresh user token using the provided refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh Token Request"
// @Success 200 {object} dto.RefreshTokenResponse "Token refreshed successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/refresh-token [post]
func (handler *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".RefreshToken")
	defer scope.End()

	req := dto.RefreshTokenRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	res, err := handler.service.RefreshToken(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to refresh token")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Token refreshed successfully")

	response.WithJSON(w, http.StatusOK, res)
}
