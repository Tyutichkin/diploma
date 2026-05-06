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
	if in.Title == "" {
		return task.Task{}, &task.ValidationError{Message: "title required"}
	}
	if in.DurationMin != nil && *in.DurationMin < 0 {
		return task.Task{}, &task.ValidationError{Message: "duration_min must not be negative"}
	}
	if err := validateWindow(
		in.WindowStartDate, in.WindowStartTime,
		in.WindowEndDate, in.WindowEndTime,
	); err != nil {
		return task.Task{}, err
	}
	fillDefaultDate(&in.WindowStartDate, in.WindowStartTime)
	fillDefaultDate(&in.WindowEndDate, in.WindowEndTime)
	return s.tasks.Create(ctx, userID, in)
}

func (s *Story) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	if err := validateWindow(
		in.WindowStartDate, in.WindowStartTime,
		in.WindowEndDate, in.WindowEndTime,
	); err != nil {
		return task.Task{}, false, err
	}
	// Для Update: дефолтную дату подставляем только если поле не nil и не "" (не сброс).
	fillDefaultDate(&in.WindowStartDate, in.WindowStartTime)
	fillDefaultDate(&in.WindowEndDate, in.WindowEndTime)
	return s.tasks.Update(ctx, userID, taskID, in)
}

// validateWindow проверяет, что если заданы обе даты+время, то start < end.
// Если указана только дата без времени — это "весь день", не валидируем время.
func validateWindow(startDate, startTime, endDate, endTime *string) error {
	sDate := ptrs.Deref(startDate)
	eDate := ptrs.Deref(endDate)
	sTime := ptrs.Deref(startTime)
	eTime := ptrs.Deref(endTime)

	if sDate == "" && eDate == "" {
		// HH:MM сравнимы лексикографически.
		if sTime != "" && eTime != "" && sTime >= eTime {
			return &task.ValidationError{Message: "window_start must be before window_end"}
		}
		return nil
	}

	if sDate == "" || eDate == "" {
		return nil
	}

	// Пустое конечное время = конец дня 23:59 (согласовано с фронтом).
	sTimeEff := sTime
	eTimeEff := eTime
	if eTimeEff == "" {
		eTimeEff = "23:59"
	}

	s, err := combineDatetime(sDate, sTimeEff)
	if err != nil {
		return nil // непарсимое — пусть БД отклонит
	}
	e, err := combineDatetime(eDate, eTimeEff)
	if err != nil {
		return nil
	}

	if !s.Before(e) {
		return &task.ValidationError{Message: "window_start must be before window_end"}
	}
	return nil
}

// combineDatetime парсит "YYYY-MM-DD" + "HH:MM"; пустое время = 00:00.
func combineDatetime(dateStr, timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Parse("2006-01-02", dateStr)
	}
	return time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, timeStr))
}

// fillDefaultDate: если время задано (непустая строка), а дата отсутствует или
// равна "" — подставляет сегодняшнюю дату. Корректно работает и для CreateInput
// (timePtr — обычный *string), и для UpdateInput (3-state *string), потому что
// nil timePtr всегда означает "не трогать".
func fillDefaultDate(datePtr **string, timePtr *string) {
	if datePtr == nil || timePtr == nil || *timePtr == "" {
		return
	}
	if *datePtr == nil || **datePtr == "" {
		today := time.Now().Format("2006-01-02")
		*datePtr = &today
	}
}

func (s *Story) CreateBatch(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	for i := range inputs {
		in := &inputs[i]
		if in.Title == "" {
			return nil, &task.ValidationError{Message: "title required"}
		}
		if in.DurationMin != nil && *in.DurationMin < 0 {
			return nil, &task.ValidationError{Message: "duration_min must not be negative"}
		}
		if err := validateWindow(in.WindowStartDate, in.WindowStartTime, in.WindowEndDate, in.WindowEndTime); err != nil {
			return nil, err
		}
		fillDefaultDate(&in.WindowStartDate, in.WindowStartTime)
		fillDefaultDate(&in.WindowEndDate, in.WindowEndTime)
	}
	return s.tasks.BatchCreate(ctx, userID, inputs)
}

func (s *Story) Delete(ctx context.Context, userID, taskID string) (bool, error) {
	return s.tasks.SoftDelete(ctx, userID, taskID)
}

// DeleteAll soft-удаляет все активные задачи пользователя одним SQL-запросом.
// Возвращает количество удалённых задач.
func (s *Story) DeleteAll(ctx context.Context, userID string) (int64, error) {
	return s.tasks.SoftDeleteAll(ctx, userID)
}

func (s *Story) Reorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return s.tasks.BulkReorder(ctx, userID, in)
}
