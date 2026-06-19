package dto

import (
	"oil/internal/domains/cinema/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateCinemaRequest struct {
	Name    string  `json:"name"              validate:"required"`
	City    string  `json:"city"              validate:"required"`
	Address *string `json:"address,omitempty"`
}

func (r *CreateCinemaRequest) ToModel(username string) model.Cinema {
	return model.Cinema{
		ID:      uuid.NewString(),
		Name:    r.Name,
		City:    r.City,
		Address: r.Address,
		Active:  true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

type UpdateCinemaRequest struct {
	Name    *string `json:"name,omitempty"    db:"name"`
	City    *string `json:"city,omitempty"    db:"city"`
	Address *string `json:"address,omitempty" db:"address"`
	Active  *bool   `json:"active,omitempty"  db:"active"`
}

type CinemaResponse struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	City    string  `json:"city"`
	Address *string `json:"address,omitempty"`
	Active  bool    `json:"active"`
	gDto.Metadata
}

func (r *CinemaResponse) FromModel(m model.Cinema) {
	r.ID = m.ID
	r.Name = m.Name
	r.City = m.City
	r.Address = m.Address
	r.Active = m.Active
	r.Metadata.FromModel(m.Metadata)
}

type GetCinemasResponse struct {
	Cinemas   []CinemaResponse `json:"cinemas"`
	TotalPage int              `json:"total_page"`
	TotalData int              `json:"total_data"`
}

func (r *GetCinemasResponse) FromModels(models []model.Cinema, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Cinemas = make([]CinemaResponse, len(models))
	for i, mod := range models {
		r.Cinemas[i].FromModel(mod)
	}
}
