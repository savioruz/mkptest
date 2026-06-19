package cinema

import (
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/cinema/model"
	"oil/internal/domains/cinema/model/dto"
	"oil/internal/domains/cinema/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Cinema
	otel    otel.Otel
}

func New(service service.Cinema, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/cinemas", func(r chi.Router) {
		r.Get("/", handler.GetAll)
		r.Post("/", handler.Create)
		r.Get("/{id}", handler.GetByID)
		r.Patch("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})
}

// Create handles cinema creation
// @Summary Create a cinema
// @Description Create a new cinema. Requires admin role.
// @Tags Cinema
// @Accept json
// @Produce json
// @Param request body dto.CreateCinemaRequest true "Create Cinema Request"
// @Success 201 {object} dto.CinemaResponse
// @Failure 400 {object} response.Error
// @Security BearerAuth
// @Router /api/cinemas [post]
func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".cinema.Create")
	defer scope.End()

	req := dto.CreateCinemaRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")
		response.WithError(w, err)

		return
	}

	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	res, err := handler.service.Create(ctx, req, userID)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create cinema")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusCreated, res)
}

// GetAll handles listing cinemas
// @Summary List cinemas
// @Description List cinemas with pagination and optional name/city search.
// @Tags Cinema
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param name query string false "Filter by name"
// @Param city query string false "Filter by city"
// @Success 200 {object} dto.GetCinemasResponse
// @Security BearerAuth
// @Router /api/cinemas [get]
func (handler *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".cinema.GetAll")
	defer scope.End()

	params := gDto.QueryParams{}
	params.FromRequest(r, true)

	filter := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: shared.SearchFieldsBuilder(r,
			shared.SearchField{Field: model.FieldName, Table: model.TableName},
			shared.SearchField{Field: model.FieldCity, Table: model.TableName},
		),
	}

	res, err := handler.service.GetAll(ctx, params, filter)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get cinemas")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// GetByID handles fetching a single cinema
// @Summary Get a cinema
// @Description Get a cinema by ID.
// @Tags Cinema
// @Produce json
// @Param id path string true "Cinema ID"
// @Success 200 {object} dto.CinemaResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/cinemas/{id} [get]
func (handler *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".cinema.GetByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	res, err := handler.service.GetByID(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get cinema")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Update handles updating a cinema
// @Summary Update a cinema
// @Description Update a cinema by ID. Requires admin role.
// @Tags Cinema
// @Accept json
// @Produce json
// @Param id path string true "Cinema ID"
// @Param request body dto.UpdateCinemaRequest true "Update Cinema Request"
// @Success 200 {object} dto.CinemaResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/cinemas/{id} [patch]
func (handler *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".cinema.Update")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateCinemaRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")
		response.WithError(w, err)

		return
	}

	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	res, err := handler.service.Update(ctx, id, req, userID)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update cinema")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Delete handles deleting a cinema
// @Summary Delete a cinema
// @Description Delete a cinema by ID. Requires admin role.
// @Tags Cinema
// @Produce json
// @Param id path string true "Cinema ID"
// @Success 200 {object} response.Message
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/cinemas/{id} [delete]
func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".cinema.Delete")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete cinema")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "cinema deleted successfully")
}
