package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	otelMocks "oil/infras/otel/mocks"
	movieModel "oil/internal/domains/movie/model"
	movieMocks "oil/internal/domains/movie/mocks"
	scheduleMocks "oil/internal/domains/schedule/mocks"
	"oil/internal/domains/schedule/model"
	"oil/internal/domains/schedule/model/dto"
	"oil/internal/domains/schedule/service"
	"oil/shared/failure"
)

func newService(t *testing.T) (service.Schedule, *scheduleMocks.MockSchedule, *movieMocks.MockMovie) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	scheduleRepo := scheduleMocks.NewMockSchedule(ctrl)
	movieRepo := movieMocks.NewMockMovie(ctrl)

	svc := service.New(scheduleRepo, movieRepo, nil, &config.Config{}, otelMocks.NewOtel())

	return svc, scheduleRepo, movieRepo
}

func TestScheduleService_Create_ComputesEndTime(t *testing.T) {
	svc, scheduleRepo, movieRepo := newService(t)

	start := time.Date(2026, 7, 1, 19, 0, 0, 0, time.UTC)

	movieRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(movieModel.Movie{ID: "movie-1", DurationMin: 120}, nil)

	var inserted model.Schedule
	scheduleRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.Schedule) error {
			inserted = m

			return nil
		})

	res, err := svc.Create(context.Background(), dto.CreateScheduleRequest{
		MovieID:   "movie-1",
		StudioID:  "studio-1",
		StartTime: start,
		Price:     50000,
	}, "admin-1")

	assert.NoError(t, err)
	assert.Equal(t, model.StatusScheduled, res.Status)
	// end_time must be start + movie duration.
	assert.Equal(t, start.Add(120*time.Minute), inserted.EndTime)
	assert.Equal(t, start, inserted.StartTime)
}

func TestScheduleService_Create_MovieNotFound(t *testing.T) {
	svc, _, movieRepo := newService(t)

	movieRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(movieModel.Movie{}, nil)

	_, err := svc.Create(context.Background(), dto.CreateScheduleRequest{
		MovieID:   "missing",
		StudioID:  "studio-1",
		StartTime: time.Now(),
		Price:     1000,
	}, "admin-1")

	assert.Error(t, err)
	assert.Equal(t, 400, failure.GetCode(err))
}

func TestScheduleService_Create_OverlapConflict(t *testing.T) {
	svc, scheduleRepo, movieRepo := newService(t)

	movieRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(movieModel.Movie{ID: "movie-1", DurationMin: 90}, nil)

	// The exclusion constraint violation must surface as a 409.
	scheduleRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).
		Return(&pq.Error{Code: "23P01"})

	_, err := svc.Create(context.Background(), dto.CreateScheduleRequest{
		MovieID:   "movie-1",
		StudioID:  "studio-1",
		StartTime: time.Now().Add(24 * time.Hour),
		Price:     1000,
	}, "admin-1")

	assert.Error(t, err)
	assert.Equal(t, 409, failure.GetCode(err))
}

func TestScheduleService_GetByID_NotFound(t *testing.T) {
	svc, scheduleRepo, _ := newService(t)

	scheduleRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(model.Schedule{}, nil)

	_, err := svc.GetByID(context.Background(), "missing")

	assert.Error(t, err)
	assert.Equal(t, 404, failure.GetCode(err))
}

func TestScheduleService_Cancel_AlreadyCancelledIsNoop(t *testing.T) {
	svc, scheduleRepo, _ := newService(t)

	scheduleRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(model.Schedule{ID: "sched-1", Status: model.StatusCancelled}, nil)
	// No Update expected — already cancelled.

	err := svc.Cancel(context.Background(), "sched-1", "admin-1")

	assert.NoError(t, err)
}
