package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
)

// routeColumns — поля в SELECT/RETURNING для таблицы routes; держим в одном месте.
const routeColumns = `id, user_id, algorithm,
	total_distance_m, total_travel_sec, total_service_sec, total_wait_sec,
	computed_at`

// scanRoute заполняет route.Route из строки запроса, выбранного по routeColumns.
func scanRoute(s interface{ Scan(...any) error }, rt *route.Route) error {
	return s.Scan(&rt.ID, &rt.UserID, &rt.Algorithm,
		&rt.TotalDistanceM, &rt.TotalTravelSec, &rt.TotalServiceSec, &rt.TotalWaitSec,
		&rt.ComputedAt)
}

type RouteRepo struct {
	pool *pgxpool.Pool
}

func NewRouteRepo(pool *pgxpool.Pool) *RouteRepo {
	return &RouteRepo{pool: pool}
}

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

	if _, err := tx.Exec(ctx, `DELETE FROM routes WHERE user_id=$1`, userID); err != nil {
		return route.Route{}, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO routes(user_id, algorithm,
			total_distance_m, total_travel_sec, total_service_sec, total_wait_sec)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+routeColumns,
		userID, algorithm, distanceM, travelSec, serviceSec, waitSec)

	var rt route.Route
	if err := scanRoute(row, &rt); err != nil {
		return route.Route{}, err
	}

	if err := insertStops(ctx, tx, rt.ID, stops); err != nil {
		return route.Route{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return route.Route{}, err
	}

	return rt, nil
}

func insertStops(ctx context.Context, tx pgx.Tx, routeID string, stops []routegate.StopInput) error {
	for _, s := range stops {
		if _, err := tx.Exec(ctx, `
			INSERT INTO route_stops(route_id, task_id, position, travel_from_prev_sec,
				arrive_time, service_start_time, service_end_time, wait_sec)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, routeID, s.TaskID, s.Position, s.TravelFromPrevSec,
			s.ArriveTime, s.ServiceStartTime, s.ServiceEndTime, s.WaitSec); err != nil {
			return err
		}
	}
	return nil
}
