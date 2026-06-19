package dto

import (
	"oil/internal/domains/movie/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateMovieRequest struct {
	Title       string  `json:"title"                  validate:"required"`
	Description *string `json:"description,omitempty"`
	DurationMin int     `json:"duration_min"           validate:"required,gt=0"`
	Genre       *string `json:"genre,omitempty"`
	Rating      *string `json:"rating,omitempty"`
	PosterURL   *string `json:"poster_url,omitempty"`
}

func (r *CreateMovieRequest) ToModel(username string) model.Movie {
	return model.Movie{
		ID:          uuid.NewString(),
		Title:       r.Title,
		Description: r.Description,
		DurationMin: r.DurationMin,
		Genre:       r.Genre,
		Rating:      r.Rating,
		PosterURL:   r.PosterURL,
		Active:      true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

// UpdateMovieRequest uses db tags so shared.TransformFields can build the
// partial-update map directly from the non-nil fields.
type UpdateMovieRequest struct {
	Title       *string `json:"title,omitempty"        db:"title"`
	Description *string `json:"description,omitempty"  db:"description"`
	DurationMin *int    `json:"duration_min,omitempty" db:"duration_min" validate:"omitempty,gt=0"`
	Genre       *string `json:"genre,omitempty"        db:"genre"`
	Rating      *string `json:"rating,omitempty"       db:"rating"`
	PosterURL   *string `json:"poster_url,omitempty"   db:"poster_url"`
	Active      *bool   `json:"active,omitempty"       db:"active"`
}

type MovieResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	DurationMin int     `json:"duration_min"`
	Genre       *string `json:"genre,omitempty"`
	Rating      *string `json:"rating,omitempty"`
	PosterURL   *string `json:"poster_url,omitempty"`
	Active      bool    `json:"active"`
	gDto.Metadata
}

func (r *MovieResponse) FromModel(m model.Movie) {
	r.ID = m.ID
	r.Title = m.Title
	r.Description = m.Description
	r.DurationMin = m.DurationMin
	r.Genre = m.Genre
	r.Rating = m.Rating
	r.PosterURL = m.PosterURL
	r.Active = m.Active
	r.Metadata.FromModel(m.Metadata)
}

type GetMoviesResponse struct {
	Movies    []MovieResponse `json:"movies"`
	TotalPage int             `json:"total_page"`
	TotalData int             `json:"total_data"`
}

func (r *GetMoviesResponse) FromModels(models []model.Movie, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Movies = make([]MovieResponse, len(models))
	for i, mod := range models {
		r.Movies[i].FromModel(mod)
	}
}
