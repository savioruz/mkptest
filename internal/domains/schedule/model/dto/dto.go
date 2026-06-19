package dto

import (
	"time"

	"oil/internal/domains/schedule/model"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateScheduleRequest struct {
	MovieID   string    `json:"movie_id"   validate:"required"`
	StudioID  string    `json:"studio_id"  validate:"required"`
	StartTime time.Time `json:"start_time" validate:"required"`
	Price     float64   `json:"price"      validate:"required,gt=0"`
}

// ToModel builds a Schedule. end_time and show_date are derived from start_time
// and the movie's duration (passed in by the service).
func (r *CreateScheduleRequest) ToModel(username string, durationMin int) model.Schedule {
	now := timezone.Now()
	start := r.StartTime
	end := start.Add(time.Duration(durationMin) * time.Minute)
	y, m, d := start.Date()
	showDate := time.Date(y, m, d, 0, 0, 0, 0, start.Location())

	return model.Schedule{
		ID:        uuid.NewString(),
		MovieID:   r.MovieID,
		StudioID:  r.StudioID,
		ShowDate:  showDate,
		StartTime: start,
		EndTime:   end,
		Price:     r.Price,
		Status:    model.StatusScheduled,
		Active:    true,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

type UpdateScheduleRequest struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	Price     *float64   `json:"price,omitempty"      validate:"omitempty,gt=0"`
	Status    *string    `json:"status,omitempty"     validate:"omitempty,oneof=scheduled cancelled finished"`
}

type ScheduleResponse struct {
	ID        string  `json:"id"`
	MovieID   string  `json:"movie_id"`
	StudioID  string  `json:"studio_id"`
	ShowDate  string  `json:"show_date"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
	Active    bool    `json:"active"`
	gDto.Metadata
}

func (r *ScheduleResponse) FromModel(m model.Schedule) {
	r.ID = m.ID
	r.MovieID = m.MovieID
	r.StudioID = m.StudioID
	r.ShowDate = m.ShowDate.Format(time.DateOnly)
	r.StartTime = m.StartTime.Format(constant.DateFormat)
	r.EndTime = m.EndTime.Format(constant.DateFormat)
	r.Price = m.Price
	r.Status = m.Status
	r.Active = m.Active
	r.Metadata.FromModel(m.Metadata)
}

type GetSchedulesResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
	TotalPage int                `json:"total_page"`
	TotalData int                `json:"total_data"`
}

func (r *GetSchedulesResponse) FromModels(models []model.Schedule, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Schedules = make([]ScheduleResponse, len(models))
	for i, mod := range models {
		r.Schedules[i].FromModel(mod)
	}
}
