package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/task"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

const taskColumns = `id, user_id, title, address_text, latitude, longitude, duration_min,
	window_start_date, window_start_time, window_end_date, window_end_time,
	sort_index, created_at, updated_at, is_completed`

func (r *TaskRepo) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE user_id=$1
		ORDER BY sort_index ASC, created_at ASC
	`, taskColumns), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]task.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TaskRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error) {
	if len(ids) == 0 {
		return []task.Task{}, nil
	}

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", i+2)
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE user_id=$1 AND id IN (%s)
	`, taskColumns, strings.Join(placeholders, ","))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TaskRepo) Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO tasks(user_id, title, address_text, latitude, longitude, duration_min,
			window_start_date, window_start_time, window_end_date, window_end_time, sort_index)
		VALUES ($1,$2,$3,$4,$5,$6,
			$7::date, $8::time, $9::date, $10::time, $11)
		RETURNING %s
	`, taskColumns),
		userID, in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin,
		nilIfEmpty(in.WindowStartDate), nilIfEmpty(in.WindowStartTime),
		nilIfEmpty(in.WindowEndDate), nilIfEmpty(in.WindowEndTime),
		in.SortIndex,
	)

	return scanTask(row)
}

func (r *TaskRepo) BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	if len(inputs) == 0 {
		return []task.Task{}, nil
	}

	query := fmt.Sprintf(`
		INSERT INTO tasks(user_id, title, address_text, latitude, longitude, duration_min,
			window_start_date, window_start_time, window_end_date, window_end_time, sort_index)
		VALUES ($1,$2,$3,$4,$5,$6,
			$7::date, $8::time, $9::date, $10::time, $11)
		RETURNING %s
	`, taskColumns)

	batch := &pgx.Batch{}
	for _, in := range inputs {
		batch.Queue(query,
			userID, in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin,
			nilIfEmpty(in.WindowStartDate), nilIfEmpty(in.WindowStartTime),
			nilIfEmpty(in.WindowEndDate), nilIfEmpty(in.WindowEndTime),
			in.SortIndex,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	out := make([]task.Task, 0, len(inputs))
	for range inputs {
		row := br.QueryRow()
		t, err := scanTask(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TaskRepo) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE tasks
		SET
			title = COALESCE($1, title),
			address_text = COALESCE($2, address_text),
			latitude = COALESCE($3, latitude),
			longitude = COALESCE($4, longitude),
			duration_min = COALESCE($5, duration_min),
			window_start_date = CASE
				WHEN $6::text IS NULL  THEN window_start_date
				WHEN $6::text = ''     THEN NULL
				ELSE $6::date
			END,
			window_start_time = CASE
				WHEN $7::text IS NULL  THEN window_start_time
				WHEN $7::text = ''     THEN NULL
				ELSE $7::time
			END,
			window_end_date = CASE
				WHEN $8::text IS NULL  THEN window_end_date
				WHEN $8::text = ''     THEN NULL
				ELSE $8::date
			END,
			window_end_time = CASE
				WHEN $9::text IS NULL  THEN window_end_time
				WHEN $9::text = ''     THEN NULL
				ELSE $9::time
			END,
			sort_index = COALESCE($10, sort_index),
			is_completed = COALESCE($11, is_completed),
			updated_at = $12
		WHERE id=$13 AND user_id=$14
		RETURNING %s
	`, taskColumns),
		in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin,
		in.WindowStartDate, in.WindowStartTime,
		in.WindowEndDate, in.WindowEndTime,
		in.SortIndex, in.IsCompleted, time.Now(),
		taskID, userID,
	)

	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.Task{}, false, nil
		}
		return task.Task{}, false, err
	}
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
	`, ids, indexes, time.Now(), userID)
	return err
}

func (r *TaskRepo) Delete(ctx context.Context, userID, taskID string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM tasks
		WHERE id=$1 AND user_id=$2
	`, taskID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *TaskRepo) DeleteAll(ctx context.Context, userID string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM routes WHERE user_id=$1`, userID); err != nil {
		return 0, err
	}

	ct, err := tx.Exec(ctx, `DELETE FROM tasks WHERE user_id=$1`, userID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// Принимает как pgx.Row, так и pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (task.Task, error) {
	var t task.Task
	var wsDate, wsTime, weDate, weTime *time.Time
	if err := s.Scan(
		&t.ID, &t.UserID, &t.Title, &t.AddressText, &t.Latitude, &t.Longitude, &t.DurationMin,
		&wsDate, &wsTime, &weDate, &weTime,
		&t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsCompleted,
	); err != nil {
		return task.Task{}, err
	}
	applyWindowFields(&t, wsDate, wsTime, weDate, weTime)
	return t, nil
}

func applyWindowFields(t *task.Task, wsDate, wsTime, weDate, weTime *time.Time) {
	if wsDate != nil {
		s := wsDate.Format("2006-01-02")
		t.WindowStartDate = &s
	}
	if wsTime != nil {
		s := wsTime.Format("15:04")
		t.WindowStartTime = &s
	}
	if weDate != nil {
		s := weDate.Format("2006-01-02")
		t.WindowEndDate = &s
	}
	if weTime != nil {
		s := weTime.Format("15:04")
		t.WindowEndTime = &s
	}
}

func nilIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
