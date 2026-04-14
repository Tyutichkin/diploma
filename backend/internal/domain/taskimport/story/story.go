// Package story реализует use case массового импорта задач из файла.
// Парсит CSV / XLSX, делегирует создание задач task.Story.CreateBatch,
// собирает ошибки построчно и возвращает сводный результат.
package story

import (
	"context"

	"planner-backend/internal/domain/task"
	taskstory "planner-backend/internal/domain/task/story"
	"planner-backend/internal/domain/taskimport"
)

// Story — use case импорта задач. Зависит только от taskstory.Story,
// т. е. не знает ни про БД, ни про HTTP — чистый domain-слой.
type Story struct {
	tasks *taskstory.Story
}

func New(tasks *taskstory.Story) *Story {
	return &Story{tasks: tasks}
}

// Result — что вернуть клиенту после импорта.
type Result struct {
	Imported []task.Task
	Errors   []taskimport.RowError
}

// Import разбирает файл data с именем filename и создаёт задачи в БД.
// startSortIndex — значение sort_index для первой созданной задачи
// (последующие получают +1, +2, …), чтобы импортированные задачи
// не конфликтовали с существующими.
func (s *Story) Import(
	ctx context.Context,
	userID string,
	data []byte,
	filename string,
	startSortIndex int,
) (Result, error) {
	format := taskimport.DetectFormat(filename)
	if format == taskimport.FormatUnknown {
		return Result{}, &task.ValidationError{Message: taskimport.UnsupportedFormatError.Error()}
	}

	parsed, err := taskimport.Parse(data, format)
	if err != nil {
		return Result{}, &task.ValidationError{Message: "failed to parse file: " + err.Error()}
	}

	if len(parsed.Rows) == 0 {
		return Result{Imported: []task.Task{}, Errors: parsed.Errors}, nil
	}

	inputs := make([]task.CreateInput, len(parsed.Rows))
	for i, row := range parsed.Rows {
		inputs[i] = task.CreateInput{
			Title:           row.Title,
			AddressText:     row.AddressText,
			DurationMin:     row.DurationMin,
			WindowStartDate: row.WindowStartDate,
			WindowStartTime: row.WindowStartTime,
			WindowEndDate:   row.WindowEndDate,
			WindowEndTime:   row.WindowEndTime,
			SortIndex:       startSortIndex + i,
		}
	}

	created, err := s.tasks.CreateBatch(ctx, userID, inputs)
	if err != nil {
		return Result{}, err
	}

	return Result{Imported: created, Errors: parsed.Errors}, nil
}
