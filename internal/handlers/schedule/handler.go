package schedule

import (
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/schedule/model"
	"oil/internal/domains/schedule/model/dto"
	"oil/internal/domains/schedule/service"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Schedule
	otel    otel.Otel
}

func New(service service.Schedule, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/schedules", func(r chi.Router) {
		r.Get("/", handler.GetAll)
		r.Post("/", handler.Create)
		r.Get("/{id}", handler.GetByID)
		r.Patch("/{id}", handler.Update)
		r.Post("/{id}/cancel", handler.Cancel)
		r.Delete("/{id}", handler.Delete)
	})
}

// Cancel cancels a schedule and triggers mass refunds
// @Summary Cancel a schedule
// @Description Cancel a schedule (cinema-initiated). Confirmed bookings are refunded asynchronously. Requires admin role.
// @Tags Schedule
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} response.Message
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules/{id}/cancel [post]
func (handler *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.Cancel")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)
	userID, _ := ctx.Value(constant.ContextKeyUserID).(string)

	if err := handler.service.Cancel(ctx, id, userID); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to cancel schedule")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "schedule cancelled; refunds are being processed")
}

// Create handles schedule creation
// @Summary Create a schedule (jadwal tayang)
// @Description Create a new show schedule. end_time is computed from the movie duration. Requires admin role.
// @Tags Schedule
// @Accept json
// @Produce json
// @Param request body dto.CreateScheduleRequest true "Create Schedule Request"
// @Success 201 {object} dto.ScheduleResponse
// @Failure 400 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules [post]
func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.Create")
	defer scope.End()

	req := dto.CreateScheduleRequest{}
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
		log.Error().Err(err).Msg("failed to create schedule")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusCreated, res)
}

// GetAll handles listing schedules
// @Summary List schedules
// @Description List schedules with pagination and optional filters (movie_id, studio_id, status, show_date).
// @Tags Schedule
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param movie_id query string false "Filter by movie"
// @Param studio_id query string false "Filter by studio"
// @Param status query string false "Filter by status"
// @Param show_date query string false "Filter by show date (YYYY-MM-DD)"
// @Success 200 {object} dto.GetSchedulesResponse
// @Security BearerAuth
// @Router /api/schedules [get]
func (handler *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.GetAll")
	defer scope.End()

	params := gDto.QueryParams{}
	params.FromRequest(r, true)

	filter := gDto.FilterGroup{Operator: gDto.FilterGroupOperatorAnd, Filters: handler.buildFilters(r)}

	res, err := handler.service.GetAll(ctx, params, filter)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get schedules")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

func (handler *Handler) buildFilters(r *http.Request) []any {
	filters := []any{}

	eq := func(param, field string) {
		if v := r.URL.Query().Get(param); v != "" {
			filters = append(filters, gDto.Filter{
				Field:    field,
				Table:    model.TableName,
				Value:    v,
				Operator: gDto.FilterOperatorEq,
			})
		}
	}

	eq(model.FieldMovieID, model.FieldMovieID)
	eq(model.FieldStudioID, model.FieldStudioID)
	eq(model.FieldStatus, model.FieldStatus)
	eq(model.FieldShowDate, model.FieldShowDate)

	return filters
}

// GetByID handles fetching a single schedule
// @Summary Get a schedule
// @Description Get a schedule by ID.
// @Tags Schedule
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} dto.ScheduleResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules/{id} [get]
func (handler *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.GetByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	res, err := handler.service.GetByID(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get schedule")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Update handles updating a schedule
// @Summary Update a schedule
// @Description Update a schedule by ID. Requires admin role.
// @Tags Schedule
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Param request body dto.UpdateScheduleRequest true "Update Schedule Request"
// @Success 200 {object} dto.ScheduleResponse
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules/{id} [patch]
func (handler *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.Update")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateScheduleRequest{}
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
		log.Error().Err(err).Msg("failed to update schedule")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Delete handles deleting a schedule
// @Summary Delete a schedule
// @Description Delete a schedule by ID. Requires admin role.
// @Tags Schedule
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} response.Message
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/schedules/{id} [delete]
func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".schedule.Delete")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete schedule")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "schedule deleted successfully")
}
