package gate

import (
	"context"

	"planner-backend/internal/domain/task"
)

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]task.Task, error)
	// GetByIDs возвращает задачи userID по списку ids; неизвестные игнорируются.
	GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error)
	Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error)
	BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error)
	Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error)
	Delete(ctx context.Context, userID, taskID string) (bool, error)
	// DeleteAll удаляет все задачи userID одним запросом.
	// Возвращает количество затронутых строк.
	DeleteAll(ctx context.Context, userID string) (int64, error)
	// BulkReorder массово обновляет sort_index у задач userID одним запросом.
	BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error
}
