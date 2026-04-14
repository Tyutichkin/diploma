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

// Колонки задач, используемые в SELECT.
const taskColumns = `id, user_id, title, address_text, latitude, longitude, duration_min,
	window_start_date, window_start_time, window_end_date, window_end_time,
	sort_index, created_at, updated_at, is_deleted, is_completed`

func (r *TaskRepo) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE user_id=$1 AND is_deleted=false
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
		WHERE user_id=$1 AND id IN (%s) AND is_deleted=false
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

	return scanTaskRow(row)
}

func (r *TaskRepo) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	// Каждое из 4 полей окна использует 3-state *string:
	//   nil    → оставить текущее (COALESCE / CASE keeps existing)
	//   &""    → очистить (set NULL)
	//   &"val" → установить новое
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
		WHERE id=$13 AND user_id=$14 AND is_deleted=false
		RETURNING %s
	`, taskColumns),
		in.Title, in.AddressText, in.Latitude, in.Longitude, in.DurationMin,
		in.WindowStartDate, in.WindowStartTime,
		in.WindowEndDate, in.WindowEndTime,
		in.SortIndex, in.IsCompleted, time.Now(),
		taskID, userID,
	)

	t, err := scanTaskRow(row)
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

// ── helpers ──────────────────────────────────────────────────────────────────

// scanTask сканирует строку из pgx.Rows в task.Task.
func scanTask(rows pgx.Rows) (task.Task, error) {
	var t task.Task
	var lat, lon *float64
	var dur *int
	var wsDate, wsTime, weDate, weTime *time.Time
	if err := rows.Scan(
		&t.ID, &t.UserID, &t.Title, &t.AddressText, &lat, &lon, &dur,
		&wsDate, &wsTime, &weDate, &weTime,
		&t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted, &t.IsCompleted,
	); err != nil {
		return task.Task{}, err
	}
	t.Latitude = lat
	t.Longitude = lon
	t.DurationMin = dur
	applyWindowFields(&t, wsDate, wsTime, weDate, weTime)
	return t, nil
}

// scanTaskRow сканирует одну строку из pgx.Row.
func scanTaskRow(row pgx.Row) (task.Task, error) {
	var t task.Task
	var lat, lon *float64
	var dur *int
	var wsDate, wsTime, weDate, weTime *time.Time
	if err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.AddressText, &lat, &lon, &dur,
		&wsDate, &wsTime, &weDate, &weTime,
		&t.SortIndex, &t.CreatedAt, &t.UpdatedAt, &t.IsDeleted, &t.IsCompleted,
	); err != nil {
		return task.Task{}, err
	}
	t.Latitude = lat
	t.Longitude = lon
	t.DurationMin = dur
	applyWindowFields(&t, wsDate, wsTime, weDate, weTime)
	return t, nil
}

// applyWindowFields конвертирует значения из БД (time.Time) в строковые поля Task.
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

// nilIfEmpty возвращает nil если указатель указывает на пустую строку.
func nilIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
