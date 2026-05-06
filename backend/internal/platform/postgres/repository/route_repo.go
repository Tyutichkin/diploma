package repository

import (
	"context"
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

// SaveOptimizedRoute перезаписывает единственный маршрут пользователя:
// существующая запись (если есть) удаляется вместе со всеми зависимостями
// (route_stops/route_stats/route_geometry — через ON DELETE CASCADE), затем
// вставляется новая. Всё в одной транзакции.
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

	if err := insertStops(ctx, tx, rt.ID, stops); err != nil {
		return route.Route{}, err
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
