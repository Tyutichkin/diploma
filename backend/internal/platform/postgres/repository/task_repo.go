package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/task"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, title, address_text, latitude, longitude, duration_min,
		       window_start, window_end, sort_index, created_at, updated_at, is_deleted
		FROM tasks
		WHERE user_id=$1 AND is_deleted=false
		ORDER BY sort_index ASC, created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]task.Task, 0)
	for rows.Next() {
		var t task.Task
		var ws, we *time.Time
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.AddressText, &t.Latitude, &t.Longitude, &t.DurationMin,
			&ws, &we, &t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted,
		); err != nil {
			return nil, err
		}

		applyTimeWindows(&t, ws, we)
		out = append(out, t)
	}
	return out, nil
}

func (r *TaskRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", i+2)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, title, address_text, latitude, longitude, duration_min,
		       window_start, window_end, sort_index, created_at, updated_at, is_deleted
		FROM tasks
		WHERE user_id=$1 AND id IN (%s) AND is_deleted=false
	`, strings.Join(placeholders, ","))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []task.Task
	for rows.Next() {
		var t task.Task
		var ws, we *time.Time
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.AddressText, &t.Latitude, &t.Longitude, &t.DurationMin,
			&ws, &we, &t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted,
		); err != nil {
			return nil, err
		}
		applyTimeWindows(&t, ws, we)
		out = append(out, t)
	}
	return out, nil
}

func (r *TaskRepo) Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error) {
	wsStr := in.WindowStart
	if wsStr != nil && *wsStr == "" {
		wsStr = nil
	}
	weStr := in.WindowEnd
	if weStr != nil && *weStr == "" {
		weStr = nil
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO tasks(user_id, title, address_text, latitude, longitude, duration_min, window_start, window_end, sort_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7::time,$8::time,$9)
		RETURNING id, user_id, title, address_text, latitude, longitude, duration_min,
		          window_start, window_end, sort_index, created_at, updated_at, is_deleted
	`, userID, in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin, wsStr, weStr, in.SortIndex)

	var t task.Task
	var ws, we *time.Time
	if err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.AddressText, &t.Latitude, &t.Longitude, &t.DurationMin,
		&ws, &we, &t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted,
	); err != nil {
		return task.Task{}, err
	}

	applyTimeWindows(&t, ws, we)
	return t, nil
}

func (r *TaskRepo) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	setWSNull := in.WindowStart != nil && *in.WindowStart == ""
	setWENull := in.WindowEnd != nil && *in.WindowEnd == ""

	row := r.pool.QueryRow(ctx, `
		UPDATE tasks
		SET
			title = COALESCE($1, title),
			address_text = COALESCE($2, address_text),
			latitude = COALESCE($3, latitude),
			longitude = COALESCE($4, longitude),
			duration_min = COALESCE($5, duration_min),
			window_start = CASE WHEN $6 THEN NULL ELSE COALESCE($7::time, window_start) END,
			window_end   = CASE WHEN $8 THEN NULL ELSE COALESCE($9::time, window_end) END,
			sort_index = COALESCE($10, sort_index),
			updated_at = $11
		WHERE id=$12 AND user_id=$13 AND is_deleted=false
		RETURNING id, user_id, title, address_text, latitude, longitude, duration_min,
		          window_start, window_end, sort_index, created_at, updated_at, is_deleted
	`,
		in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin,
		setWSNull, in.WindowStart,
		setWENull, in.WindowEnd,
		in.SortIndex, time.Now(),
		taskID, userID,
	)

	var t task.Task
	var ws, we *time.Time
	if err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.AddressText, &t.Latitude, &t.Longitude, &t.DurationMin,
		&ws, &we, &t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted,
	); err != nil {
		return task.Task{}, false, nil
	}

	applyTimeWindows(&t, ws, we)
	return t, true, nil
}

func (r *TaskRepo) BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error {
	if len(in.Items) == 0 {
		return nil
	}

	ids := make([]string, len(in.Items))
	indexes := make([]int, len(in.Items))
	for i, item := range in.Items {
		ids[i] = item.TaskID
		indexes[i] = item.SortIndex
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE tasks
		SET sort_index = v.sort_index,
		    updated_at = $3
		FROM (
			SELECT unnest($1::uuid[]) AS id,
			       unnest($2::int[])  AS sort_index
		) AS v
		WHERE tasks.id = v.id
		  AND tasks.user_id = $4
		  AND tasks.is_deleted = false
	`, ids, indexes, time.Now(), userID)
	return err
}

func (r *TaskRepo) SoftDelete(ctx context.Context, userID, taskID string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		UPDATE tasks SET is_deleted=true, updated_at=$1
		WHERE id=$2 AND user_id=$3 AND is_deleted=false
	`, time.Now(), taskID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func applyTimeWindows(t *task.Task, ws, we *time.Time) {
	if ws != nil {
		s := ws.Format("15:04:05")
		t.WindowStart = &s
	}
	if we != nil {
		s := we.Format("15:04:05")
		t.WindowEnd = &s
	}
}
