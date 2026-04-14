package gate

import (
	"context"

	"planner-backend/internal/domain/task"
)

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]task.Task, error)
	// GetByIDs returns the tasks with the given IDs that belong to userID.
	// Unknown or deleted IDs are silently ignored.
	GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error)
	Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error)
	BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error)
	Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error)
	SoftDelete(ctx context.Context, userID, taskID string) (bool, error)
	// BulkReorder sets sort_index for multiple tasks belonging to userID in a single query.
	BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error
}
