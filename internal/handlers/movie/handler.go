package movie

import (
	"net/http"

	"oil/infras/otel"
	"oil/internal/domains/movie/model"
	"oil/internal/domains/movie/model/dto"
	"oil/internal/domains/movie/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Movie
	otel    otel.Otel
}

func New(service service.Movie, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/movies", func(r chi.Router) {
		r.Get("/", handler.GetAll)
		r.Post("/", handler.Create)
		r.Get("/{id}", handler.GetByID)
		r.Patch("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})
}

// Create handles movie creation
// @Summary Create a movie
// @Description Create a new movie. Requires admin role.
// @Tags Movie
// @Accept json
// @Produce json
// @Param request body dto.CreateMovieRequest true "Create Movie Request"
// @Success 201 {object} dto.MovieResponse
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Security BearerAuth
// @Router /api/movies [post]
func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".movie.Create")
	defer scope.End()

	req := dto.CreateMovieRequest{}
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
		log.Error().Err(err).Msg("failed to create movie")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusCreated, res)
}

// GetAll handles listing movies
// @Summary List movies
// @Description List movies with pagination and optional title search.
// @Tags Movie
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param sort_by query string false "Sort by"
// @Param sort_dir query string false "Sort direction"
// @Param title query string false "Filter by title"
// @Success 200 {object} dto.GetMoviesResponse
// @Failure 500 {object} response.Error
// @Security BearerAuth
// @Router /api/movies [get]
func (handler *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".movie.GetAll")
	defer scope.End()

	params := gDto.QueryParams{}
	params.FromRequest(r, true)

	filter := gDto.FilterGroup{
		Filters: shared.SearchFieldsBuilder(r,
			shared.SearchField{Field: model.FieldTitle, Table: model.TableName},
		),
	}

	res, err := handler.service.GetAll(ctx, params, filter)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get movies")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// GetByID handles fetching a single movie
// @Summary Get a movie
// @Description Get a movie by ID.
// @Tags Movie
// @Produce json
// @Param id path string true "Movie ID"
// @Success 200 {object} dto.MovieResponse
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/movies/{id} [get]
func (handler *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".movie.GetByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	res, err := handler.service.GetByID(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get movie")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Update handles updating a movie
// @Summary Update a movie
// @Description Update a movie by ID. Requires admin role.
// @Tags Movie
// @Accept json
// @Produce json
// @Param id path string true "Movie ID"
// @Param request body dto.UpdateMovieRequest true "Update Movie Request"
// @Success 200 {object} dto.MovieResponse
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Security BearerAuth
// @Router /api/movies/{id} [patch]
func (handler *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".movie.Update")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateMovieRequest{}
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
		log.Error().Err(err).Msg("failed to update movie")
		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}

// Delete handles deleting a movie
// @Summary Delete a movie
// @Description Delete a movie by ID. Requires admin role.
// @Tags Movie
// @Produce json
// @Param id path string true "Movie ID"
// @Success 200 {object} response.Message
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Security BearerAuth
// @Router /api/movies/{id} [delete]
func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".movie.Delete")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete movie")
		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "movie deleted successfully")
}
