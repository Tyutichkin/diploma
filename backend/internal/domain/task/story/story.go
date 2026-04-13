package story

import (
	"context"
	"errors"
	"strconv"
	"strings"

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
	if err := validateTimeWindow(in.WindowStart, in.WindowEnd); err != nil {
		return task.Task{}, err
	}
	return s.tasks.Create(ctx, userID, in)
}

func (s *Story) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	if err := validateTimeWindow(in.WindowStart, in.WindowEnd); err != nil {
		return task.Task{}, false, err
	}
	return s.tasks.Update(ctx, userID, taskID, in)
}

// validateTimeWindow returns an error when both boundaries are set and start >= end.
// A single boundary (only start or only end) is accepted without validation.
func validateTimeWindow(start, end *string) error {
	if start == nil || end == nil || *start == "" || *end == "" {
		return nil
	}
	sMin := parseWindowTimeMins(*start)
	eMin := parseWindowTimeMins(*end)
	if sMin < 0 || eMin < 0 {
		return nil // unparseable values — let the DB reject them
	}
	if sMin >= eMin {
		return errors.New("window_start must be before window_end")
	}
	return nil
}

// parseWindowTimeMins converts "HH:MM" or "HH:MM:SS" to minutes since midnight.
// Returns -1 for nil or unparseable input.
func parseWindowTimeMins(s string) int {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return -1
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return -1
	}
	return h*60 + m
}

func (s *Story) Delete(ctx context.Context, userID, taskID string) (bool, error) {
	return s.tasks.SoftDelete(ctx, userID, taskID)
}

func (s *Story) Reorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return s.tasks.BulkReorder(ctx, userID, in)
}
