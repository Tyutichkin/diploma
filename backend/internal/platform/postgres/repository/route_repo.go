package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
)

// routeColumns — поля в SELECT/RETURNING для таблицы routes; держим в одном месте.
const routeColumns = `id, user_id, status, source, name, algorithm, started_at, finished_at, created_at`

// scanRoute заполняет route.Route из строки запроса, выбранного по routeColumns.
func scanRoute(s interface{ Scan(...any) error }, rt *route.Route) error {
	return s.Scan(&rt.ID, &rt.UserID, &rt.Status, &rt.Source, &rt.Name,
		&rt.Algorithm, &rt.StartedAt, &rt.FinishedAt, &rt.CreatedAt)
}

type RouteRepo struct {
	pool *pgxpool.Pool
}

func NewRouteRepo(pool *pgxpool.Pool) *RouteRepo {
	return &RouteRepo{pool: pool}
}

// deleteUserRoutes удаляет все существующие маршруты пользователя.
// Используется перед сохранением нового, поскольку у нас политика "один маршрут на пользователя".
func deleteUserRoutes(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM routes WHERE user_id=$1`, userID)
	return err
}

func (r *RouteRepo) CreateDraft(ctx context.Context, userID, source string) (route.Route, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return route.Route{}, err
	}
	defer tx.Rollback(ctx)

	if err := deleteUserRoutes(ctx, tx, userID); err != nil {
		return route.Route{}, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO routes(user_id, status, source)
		VALUES ($1, 'draft', $2)
		RETURNING `+routeColumns,
		userID, source)

	var rt route.Route
	if err := scanRoute(row, &rt); err != nil {
		return route.Route{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return route.Route{}, err
	}
	return rt, nil
}

func (r *RouteRepo) ListByUser(ctx context.Context, userID string) ([]route.Route, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+routeColumns+`
		FROM routes
		WHERE user_id=$1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]route.Route, 0)
	for rows.Next() {
		var rt route.Route
		if err := scanRoute(rows, &rt); err != nil {
			return nil, err
		}
		out = append(out, rt)
	}
	return out, nil
}

func (r *RouteRepo) ReplaceStops(ctx context.Context, routeID string, taskIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM route_stops WHERE route_id=$1`, routeID); err != nil {
		return err
	}

	for i, taskID := range taskIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO route_stops(route_id, task_id, position)
			VALUES ($1,$2,$3)
		`, routeID, taskID, i); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *RouteRepo) GetFull(ctx context.Context, userID, routeID string) (route.Full, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+routeColumns+`
		FROM routes
		WHERE id=$1 AND user_id=$2
	`, routeID, userID)

	var rt route.Route
	if err := scanRoute(row, &rt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return route.Full{}, false, nil
		}
		return route.Full{}, false, err
	}

	stops, err := r.fetchStops(ctx, routeID)
	if err != nil {
		return route.Full{}, false, err
	}

	stats, err := r.fetchStats(ctx, routeID)
	if err != nil {
		return route.Full{}, false, err
	}

	geom, err := r.fetchGeometry(ctx, routeID)
	if err != nil {
		return route.Full{}, false, err
	}

	return route.Full{Route: rt, Stops: stops, Stats: stats, Geometry: geom}, true, nil
}

func (r *RouteRepo) fetchStops(ctx context.Context, routeID string) ([]route.Stop, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, route_id, task_id, position, travel_from_prev_sec, arrive_time,
		       service_start_time, service_end_time, wait_sec
		FROM route_stops
		WHERE route_id=$1
		ORDER BY position ASC
	`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stops := make([]route.Stop, 0)
	for rows.Next() {
		var s route.Stop
		if err := rows.Scan(
			&s.ID, &s.RouteID, &s.TaskID, &s.Position,
			&s.TravelFromPrevSec, &s.ArriveTime,
			&s.ServiceStartTime, &s.ServiceEndTime, &s.WaitSec,
		); err != nil {
			return nil, err
		}
		stops = append(stops, s)
	}
	return stops, nil
}

func (r *RouteRepo) fetchStats(ctx context.Context, routeID string) (*route.Stats, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT route_id, total_distance_m, total_travel_sec, total_service_sec, total_wait_sec, computed_at
		FROM route_stats WHERE route_id=$1
	`, routeID)

	var st route.Stats
	if err := row.Scan(&st.RouteID, &st.TotalDistanceM, &st.TotalTravelSec, &st.TotalServiceSec, &st.TotalWaitSec, &st.ComputedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &st, nil
}

func (r *RouteRepo) fetchGeometry(ctx context.Context, routeID string) (*route.Geometry, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT route_id, polyline, bbox_json::text, geojson_json::text, updated_at
		FROM route_geometry WHERE route_id=$1
	`, routeID)

	var geom route.Geometry
	if err := row.Scan(&geom.RouteID, &geom.Polyline, &geom.BBoxJSON, &geom.GeoJSON, &geom.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &geom, nil
}

func (r *RouteRepo) MarkFailed(ctx context.Context, routeID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE routes SET status='failed', finished_at=$1
		WHERE id=$2
	`, now, routeID)
	return err
}

func (r *RouteRepo) DeleteRoute(ctx context.Context, userID, routeID string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM routes WHERE id=$1 AND user_id=$2
	`, routeID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *RouteRepo) DeleteAllRoutes(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM routes WHERE user_id=$1`, userID)
	return err
}

func (r *RouteRepo) RenameRoute(ctx context.Context, userID, routeID, name string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		UPDATE routes SET name=$1 WHERE id=$2 AND user_id=$3
	`, name, routeID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// SaveOptimizedRoute сохраняет маршрут целиком (запись + остановки + статистика) в одной транзакции.
func (r *RouteRepo) SaveOptimizedRoute(
	ctx context.Context,
	userID, algorithm string,
	stops []routegate.StopInput,
	distanceM, travelSec, serviceSec, waitSec int,
) (route.Route, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return route.Route{}, err
	}
	defer tx.Rollback(ctx)

	if err := deleteUserRoutes(ctx, tx, userID); err != nil {
		return route.Route{}, err
	}

	now := time.Now()
	row := tx.QueryRow(ctx, `
		INSERT INTO routes(user_id, status, source, algorithm, started_at, finished_at)
		VALUES ($1, 'optimized', 'optimized', $2, $3, $3)
		RETURNING `+routeColumns,
		userID, algorithm, now)

	var rt route.Route
	if err := scanRoute(row, &rt); err != nil {
		return route.Route{}, err
	}

	for _, s := range stops {
		if _, err := tx.Exec(ctx, `
			INSERT INTO route_stops(route_id, task_id, position, travel_from_prev_sec,
				arrive_time, service_start_time, service_end_time, wait_sec)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, rt.ID, s.TaskID, s.Position, s.TravelFromPrevSec,
			s.ArriveTime, s.ServiceStartTime, s.ServiceEndTime, s.WaitSec); err != nil {
			return route.Route{}, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO route_stats(route_id, total_distance_m, total_travel_sec, total_service_sec, total_wait_sec)
		VALUES ($1, $2, $3, $4, $5)
	`, rt.ID, distanceM, travelSec, serviceSec, waitSec); err != nil {
		return route.Route{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return route.Route{}, err
	}

	return rt, nil
}
