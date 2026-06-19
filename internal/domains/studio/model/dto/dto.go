package dto

import (
	"oil/internal/domains/studio/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateStudioRequest struct {
	CinemaID   string `json:"cinema_id"    validate:"required"`
	Name       string `json:"name"         validate:"required"`
	RowCount   int    `json:"row_count"    validate:"required,gt=0,lte=26"`
	ColsPerRow int    `json:"cols_per_row" validate:"required,gt=0,lte=50"`
	// VipRows marks the last N rows as VIP seating (optional, 0 = none).
	VipRows int `json:"vip_rows" validate:"omitempty,gte=0"`
}

func (r *CreateStudioRequest) ToModel(username string) model.Studio {
	return model.Studio{
		ID:         uuid.NewString(),
		CinemaID:   r.CinemaID,
		Name:       r.Name,
		TotalSeats: r.RowCount * r.ColsPerRow,
		RowCount:   r.RowCount,
		ColsPerRow: r.ColsPerRow,
		Active:     true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

type UpdateStudioRequest struct {
	Name   *string `json:"name,omitempty"   db:"name"`
	Active *bool   `json:"active,omitempty" db:"active"`
}

type StudioResponse struct {
	ID         string `json:"id"`
	CinemaID   string `json:"cinema_id"`
	Name       string `json:"name"`
	TotalSeats int    `json:"total_seats"`
	RowCount   int    `json:"row_count"`
	ColsPerRow int    `json:"cols_per_row"`
	Active     bool   `json:"active"`
	gDto.Metadata
}

func (r *StudioResponse) FromModel(m model.Studio) {
	r.ID = m.ID
	r.CinemaID = m.CinemaID
	r.Name = m.Name
	r.TotalSeats = m.TotalSeats
	r.RowCount = m.RowCount
	r.ColsPerRow = m.ColsPerRow
	r.Active = m.Active
	r.Metadata.FromModel(m.Metadata)
}

type GetStudiosResponse struct {
	Studios   []StudioResponse `json:"studios"`
	TotalPage int              `json:"total_page"`
	TotalData int              `json:"total_data"`
}

func (r *GetStudiosResponse) FromModels(models []model.Studio, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Studios = make([]StudioResponse, len(models))
	for i, mod := range models {
		r.Studios[i].FromModel(mod)
	}
}
