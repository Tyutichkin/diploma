package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
)

type RouteRepo struct {
	pool *pgxpool.Pool
}

func NewRouteRepo(pool *pgxpool.Pool) *RouteRepo {
	return &RouteRepo{pool: pool}
}

func (r *RouteRepo) CreateDraft(ctx context.Context, userID, source string) (route.Route, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO routes(user_id, status, source)
		VALUES ($1, 'draft', $2)
		RETURNING id, user_id, status, source, name, algorithm, started_at, finished_at, created_at
	`, userID, source)

	var rt route.Route
	if err := row.Scan(&rt.ID, &rt.UserID, &rt.Status, &rt.Source, &rt.Name, &rt.Algorithm, &rt.StartedAt, &rt.FinishedAt, &rt.CreatedAt); err != nil {
		return route.Route{}, err
	}
	return rt, nil
}

func (r *RouteRepo) ListByUser(ctx context.Context, userID string) ([]route.Route, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, status, source, name, algorithm, started_at, finished_at, created_at
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
		if err := rows.Scan(&rt.ID, &rt.UserID, &rt.Status, &rt.Source, &rt.Name, &rt.Algorithm, &rt.StartedAt, &rt.FinishedAt, &rt.CreatedAt); err != nil {
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
		SELECT id, user_id, status, source, name, algorithm, started_at, finished_at, created_at
		FROM routes
		WHERE id=$1 AND user_id=$2
	`, routeID, userID)

	var rt route.Route
	if err := row.Scan(&rt.ID, &rt.UserID, &rt.Status, &rt.Source, &rt.Name, &rt.Algorithm, &rt.StartedAt, &rt.FinishedAt, &rt.CreatedAt); err != nil {
		return route.Full{}, false, nil
	}

	srows, err := r.pool.Query(ctx, `
		SELECT id, route_id, task_id, position, travel_from_prev_sec, arrive_time,
		       service_start_time, service_end_time, wait_sec
		FROM route_stops
		WHERE route_id=$1
		ORDER BY position ASC
	`, routeID)
	if err != nil {
		return route.Full{}, false, err
	}
	defer srows.Close()

	stops := make([]route.Stop, 0)
	for srows.Next() {
		var s route.Stop
		if err := srows.Scan(
			&s.ID, &s.RouteID, &s.TaskID, &s.Position,
			&s.TravelFromPrevSec, &s.ArriveTime,
			&s.ServiceStartTime, &s.ServiceEndTime, &s.WaitSec,
		); err != nil {
			return route.Full{}, false, err
		}
		stops = append(stops, s)
	}

	stRow := r.pool.QueryRow(ctx, `
		SELECT route_id, total_distance_m, total_travel_sec, total_service_sec, total_wait_sec, computed_at
		FROM route_stats WHERE route_id=$1
	`, routeID)

	var st route.Stats
	if err := stRow.Scan(&st.RouteID, &st.TotalDistanceM, &st.TotalTravelSec, &st.TotalServiceSec, &st.TotalWaitSec, &st.ComputedAt); err != nil {
		st = route.Stats{}
	}

	gRow := r.pool.QueryRow(ctx, `
		SELECT route_id, polyline, bbox_json::text, geojson_json::text, updated_at
		FROM route_geometry WHERE route_id=$1
	`, routeID)

	var geom route.Geometry
	if err := gRow.Scan(&geom.RouteID, &geom.Polyline, &geom.BBoxJSON, &geom.GeoJSON, &geom.UpdatedAt); err != nil {
		geom = route.Geometry{}
	}

	var stPtr *route.Stats
	if st.RouteID != "" {
		stPtr = &st
	}
	var geomPtr *route.Geometry
	if geom.RouteID != "" {
		geomPtr = &geom
	}

	return route.Full{Route: rt, Stops: stops, Stats: stPtr, Geometry: geomPtr}, true, nil
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

// SaveOptimizedRoute creates a fully optimised route (record + stops + stats)
// inside a single transaction and returns the created route.
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

	now := time.Now()
	row := tx.QueryRow(ctx, `
		INSERT INTO routes(user_id, status, source, algorithm, started_at, finished_at)
		VALUES ($1, 'optimized', 'optimized', $2, $3, $3)
		RETURNING id, user_id, status, source, name, algorithm, started_at, finished_at, created_at
	`, userID, algorithm, now)

	var rt route.Route
	if err := row.Scan(
		&rt.ID, &rt.UserID, &rt.Status, &rt.Source, &rt.Name,
		&rt.Algorithm, &rt.StartedAt, &rt.FinishedAt, &rt.CreatedAt,
	); err != nil {
		return route.Route{}, err
	}

	for _, s := range stops {
		if _, err := tx.Exec(ctx, `
			INSERT INTO route_stops(route_id, task_id, position, travel_from_prev_sec)
			VALUES ($1, $2, $3, $4)
		`, rt.ID, s.TaskID, s.Position, s.TravelFromPrevSec); err != nil {
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
