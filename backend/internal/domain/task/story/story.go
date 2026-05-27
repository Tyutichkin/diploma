package story

import (
	"context"
	"fmt"
	"time"

	"planner-backend/internal/common/ptrs"
	"planner-backend/internal/domain/task"
	taskgate "planner-backend/internal/domain/task/gate"
)

type Story struct {
	tasks taskgate.Repository
}

func New(tasks taskgate.Repository) *Story {
	return &Story{tasks: tasks}
}

func (s *Story) List(ctx context.Context, userID string) ([]task.Task, error) {
	return s.tasks.ListByUser(ctx, userID)
}

func (s *Story) Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error) {
	if err := validateCreate(&in); err != nil {
		return task.Task{}, err
	}
	return s.tasks.Create(ctx, userID, in)
}

func (s *Story) CreateBatch(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	for i := range inputs {
		if err := validateCreate(&inputs[i]); err != nil {
			return nil, err
		}
	}
	return s.tasks.BatchCreate(ctx, userID, inputs)
}

func (s *Story) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	if in.Window != nil {
		if err := validateWindow(*in.Window); err != nil {
			return task.Task{}, false, err
		}
		fillDefaultDates(in.Window)
	}
	return s.tasks.Update(ctx, userID, taskID, in)
}

func (s *Story) Delete(ctx context.Context, userID, taskID string) (bool, error) {
	return s.tasks.Delete(ctx, userID, taskID)
}

func (s *Story) DeleteAll(ctx context.Context, userID string) (int64, error) {
	return s.tasks.DeleteAll(ctx, userID)
}

func (s *Story) Reorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return s.tasks.BulkReorder(ctx, userID, in)
}

func validateCreate(in *task.CreateInput) error {
	if in.Title == "" {
		return &task.ValidationError{Message: "title required"}
	}
	if in.DurationMin != nil && *in.DurationMin < 0 {
		return &task.ValidationError{Message: "duration_min must not be negative"}
	}
	if err := validateWindow(in.Window); err != nil {
		return err
	}
	fillDefaultDates(&in.Window)
	return nil
}

func validateWindow(w task.TimeWindow) error {
	sDate := ptrs.Deref(w.StartDate)
	eDate := ptrs.Deref(w.EndDate)
	sTime := ptrs.Deref(w.StartTime)
	eTime := ptrs.Deref(w.EndTime)

	if sDate == "" && eDate == "" {
		if sTime != "" && eTime != "" && sTime >= eTime {
			return &task.ValidationError{Message: "window_start must be before window_end"}
		}
		return nil
	}

	if sDate == "" || eDate == "" {
		return nil
	}

	if eTime == "" {
		eTime = "23:59"
	}

	start, err := combineDatetime(sDate, sTime)
	if err != nil {
		return nil
	}
	end, err := combineDatetime(eDate, eTime)
	if err != nil {
		return nil
	}

	if !start.Before(end) {
		return &task.ValidationError{Message: "window_start must be before window_end"}
	}
	return nil
}

func combineDatetime(dateStr, timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Parse("2006-01-02", dateStr)
	}
	return time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, timeStr))
}

// fillDefaultDates подставляет сегодняшнюю дату, если время задано без даты —
// удобство для UI «создаю задачу на сегодня, ввожу только время».
func fillDefaultDates(w *task.TimeWindow) {
	defaultDateIfTimeSet(&w.StartDate, w.StartTime)
	defaultDateIfTimeSet(&w.EndDate, w.EndTime)
}

func defaultDateIfTimeSet(datePtr **string, timePtr *string) {
	if timePtr == nil || *timePtr == "" {
		return
	}
	if *datePtr == nil || **datePtr == "" {
		today := time.Now().Format("2006-01-02")
		*datePtr = &today
	}
}
