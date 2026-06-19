package studio

import (
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/studio/model"
	"oil/internal/domains/studio/model/dto"
	"oil/internal/domains/studio/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Studio
	otel    otel.Otel
}

func New(service service.Studio, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/studios", func(r chi.Router) {
		r.Get("/", handler.GetAll)
		r.Post("/", handler.Create)
		r.Get("/{id}", handler.GetByID)
		r.Patch("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})
}

// Create handles studio creation (auto-generates seats)
// @Summary Create a studio
// @Description Create a studio and auto-generate its seat grid. Requires admin role.
// @Tags Studio
// @Accept json
// @Produce json
// @Param request body dto.CreateStudioRequest true "Create Studio Request"
// @Success 201 {object} dto.StudioResponse
// @Failure 400 {object} response.Error
// @Security BearerAuth
// @Router /api/studios [post]
func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".studio.Create")
	defer scope.End()

	req := dto.CreateStudioRequest{}
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
		log.Error().Err(err).Msg("failed to create studio")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusCreated, res)
}

// GetAll handles listing studios
// @Summary List studios
// @Description List studios with pagination and optional name search.
// @Tags Studio
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param name query string false "Filter by name"
// @Success 200 {object} dto.GetStudiosResponse
// @Security BearerAuth
// @Router /api/studios [get]
func (handler *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".studio.GetAll")
	defer scope.End()

	params := gDto.QueryParams{}
	params.FromRequest(r, true)

	filter := gDto.FilterGroup{
		Filters: shared.SearchFieldsBuilder(r,
			shared.SearchField{Field: model.FieldName, Table: model.TableName},
		),
	}

	res, err := handler.service.GetAll(ctx, params, filter)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get studios")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// GetByID handles fetching a single studio
// @Summary Get a studio
// @Description Get a studio by ID.
// @Tags Studio
// @Produce json
// @Param id path string true "Studio ID"
// @Success 200 {object} dto.StudioResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/studios/{id} [get]
func (handler *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".studio.GetByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	res, err := handler.service.GetByID(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get studio")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Update handles updating a studio
// @Summary Update a studio
// @Description Update a studio by ID. Requires admin role.
// @Tags Studio
// @Accept json
// @Produce json
// @Param id path string true "Studio ID"
// @Param request body dto.UpdateStudioRequest true "Update Studio Request"
// @Success 200 {object} dto.StudioResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/studios/{id} [patch]
func (handler *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".studio.Update")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateStudioRequest{}
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
		log.Error().Err(err).Msg("failed to update studio")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Delete handles deleting a studio
// @Summary Delete a studio
// @Description Delete a studio by ID (cascades seats). Requires admin role.
// @Tags Studio
// @Produce json
// @Param id path string true "Studio ID"
// @Success 200 {object} response.Message
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/studios/{id} [delete]
func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".studio.Delete")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete studio")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "studio deleted successfully")
}
