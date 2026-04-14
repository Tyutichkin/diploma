package story

import (
	"context"
	"fmt"
	"time"

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
	// Если дата не указана — проставляем сегодня.
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
	// Для update: заполняем дефолтную дату только если поле присутствует (не nil)
	// и не является сбросом (не "").
	fillDefaultDateUpdate(&in.WindowStartDate, in.WindowStartTime)
	fillDefaultDateUpdate(&in.WindowEndDate, in.WindowEndTime)
	return s.tasks.Update(ctx, userID, taskID, in)
}

// validateWindow проверяет, что если заданы обе даты+время, то start < end.
// Если указана только дата без времени — это "весь день", не валидируем время.
func validateWindow(startDate, startTime, endDate, endTime *string) error {
	sDate := derefStr(startDate)
	eDate := derefStr(endDate)
	sTime := derefStr(startTime)
	eTime := derefStr(endTime)

	// Обе даты пусты
	if sDate == "" && eDate == "" {
		// Если указаны оба времени — проверяем порядок на одном дне (HH:MM сравнимы лексикографически)
		if sTime != "" && eTime != "" && sTime >= eTime {
			return &task.ValidationError{Message: "window_start must be before window_end"}
		}
		return nil
	}

	// Если только одна граница — допустимо, не валидируем перекрытие
	if sDate == "" || eDate == "" {
		return nil
	}

	s, err := combineDatetime(sDate, sTime)
	if err != nil {
		return nil // непарсимое — пусть БД отклонит
	}
	e, err := combineDatetime(eDate, eTime)
	if err != nil {
		return nil
	}

	if !s.Before(e) {
		return &task.ValidationError{Message: "window_start must be before window_end"}
	}
	return nil
}

// combineDatetime собирает time.Time из строки даты "YYYY-MM-DD" и времени "HH:MM".
// Если время пустое — считаем начало дня (00:00).
func combineDatetime(dateStr, timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Parse("2006-01-02", dateStr)
	}
	return time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, timeStr))
}

// fillDefaultDate: если время задано, а дата нет — ставим сегодня.
func fillDefaultDate(datePtr **string, timePtr *string) {
	if datePtr == nil || timePtr == nil {
		return
	}
	// Время задано, дата пуста → ставим сегодня
	if *timePtr != "" && (*datePtr == nil || **datePtr == "") {
		today := time.Now().Format("2006-01-02")
		*datePtr = &today
	}
}

// fillDefaultDateUpdate: для UpdateInput (3-state *string).
// Ставим дефолтную дату, если время присутствует (не nil, не ""), а дата нет.
func fillDefaultDateUpdate(datePtr **string, timePtr *string) {
	if timePtr == nil {
		return
	}
	// Время задано — убеждаемся, что дата тоже задана
	if *timePtr != "" {
		if datePtr != nil && (*datePtr == nil || **datePtr == "") {
			today := time.Now().Format("2006-01-02")
			*datePtr = &today
		}
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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

func (s *Story) Reorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return s.tasks.BulkReorder(ctx, userID, in)
}
