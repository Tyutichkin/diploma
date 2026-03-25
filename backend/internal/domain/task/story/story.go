package story

import (
	"context"
	"errors"

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
	if in.Title == "" || in.AddressText == "" {
		return task.Task{}, errors.New("title/address required")
	}
	if in.DurationMin <= 0 {
		return task.Task{}, errors.New("duration_min must be positive")
	}
	return s.tasks.Create(ctx, userID, in)
}

func (s *Story) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	return s.tasks.Update(ctx, userID, taskID, in)
}

func (s *Story) Delete(ctx context.Context, userID, taskID string) (bool, error) {
	return s.tasks.SoftDelete(ctx, userID, taskID)
}

func (s *Story) Reorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return s.tasks.BulkReorder(ctx, userID, in)
}
