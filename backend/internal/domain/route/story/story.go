package story

import (
	"context"
	"errors"
	"fmt"
	"time"

	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
	routeopt "planner-backend/internal/domain/route/optimizer"
	taskgate "planner-backend/internal/domain/task/gate"
	distancepkg "planner-backend/internal/platform/distance"
)

type Story struct {
	routes       routegate.Repository
	tasks        taskgate.Repository
	distProvider distancepkg.Provider
	optimizer    routeopt.Optimizer
}

func New(
	routes routegate.Repository,
	tasks taskgate.Repository,
	dist distancepkg.Provider,
	opt routeopt.Optimizer,
) *Story {
	return &Story{
		routes:       routes,
		tasks:        tasks,
		distProvider: dist,
		optimizer:    opt,
	}
}

// CreateDraft saves a manually ordered list of task IDs as a draft route.
func (s *Story) CreateDraft(ctx context.Context, userID, source string, orderedTaskIDs []string) (route.Route, error) {
	if source == "" {
		source = "manual"
	}

	rt, err := s.routes.CreateDraft(ctx, userID, source)
	if err != nil {
		return route.Route{}, err
	}

	if len(orderedTaskIDs) > 0 {
		if err := s.routes.ReplaceStops(ctx, rt.ID, orderedTaskIDs); err != nil {
			_ = s.routes.MarkFailed(ctx, rt.ID)
			return route.Route{}, err
		}
	}

	return rt, nil
}

// PrecedenceConstraint задаёт попарное ограничение порядка посещения задач:
// задача BeforeTaskID должна быть выполнена строго до задачи AfterTaskID.
type PrecedenceConstraint struct {
	BeforeTaskID string
	AfterTaskID  string
}

// Optimize строит маршрут: получает матрицу расстояний (от фронта или через OSRM),
// запускает алгоритм оптимизации с учётом временны́х окон и сохраняет результат.
//
// externalMatrix[i][j] должна соответствовать порядку задач из GetByIDs.
// Если nil — матрица запрашивается у настроенного distance.Provider (OSRM).
//
// startTaskID и endTaskID фиксируют первую и последнюю точку маршрута соответственно.
// precedences задаёт попарные ограничения порядка посещения.
func (s *Story) Optimize(
	ctx context.Context,
	userID string,
	taskIDs []string,
	startTimeUnix int64,
	externalMatrix [][]distancepkg.Edge,
	startTaskID, endTaskID *string,
	precedences []PrecedenceConstraint,
) (route.Full, error) {
	if len(taskIDs) < 2 {
		return route.Full{}, errors.New("at least 2 tasks required for optimisation")
	}

	if startTaskID != nil && endTaskID != nil && *startTaskID == *endTaskID {
		return route.Full{}, errors.New("start and end task must be different")
	}

	tasks, err := s.tasks.GetByIDs(ctx, userID, taskIDs)
	if err != nil {
		return route.Full{}, fmt.Errorf("fetch tasks: %w", err)
	}
	if len(tasks) < 2 {
		return route.Full{}, errors.New("at least 2 valid tasks required")
	}

	// Если фронт прислал матрицу (Yandex Maps) — берём её; иначе запрашиваем OSRM.
	var matrix [][]distancepkg.Edge
	if len(externalMatrix) > 0 {
		matrix = externalMatrix
	} else {
		points := make([]distancepkg.Point, len(tasks))
		for i, t := range tasks {
			if t.Latitude == nil || t.Longitude == nil {
				return route.Full{}, fmt.Errorf("task %q has no coordinates: required for distance matrix", t.ID)
			}
			points[i] = distancepkg.Point{Lat: *t.Latitude, Lng: *t.Longitude}
		}
		matrix, err = s.distProvider.GetMatrix(ctx, points)
		if err != nil {
			return route.Full{}, fmt.Errorf("distance matrix: %w", err)
		}
	}

	nodes := make([]routeopt.Node, len(tasks))
	taskIDToIdx := make(map[string]int, len(tasks))
	for i, t := range tasks {
		nodes[i] = routeopt.Node{
			TaskID:      t.ID,
			Lat:         derefF64(t.Latitude),
			Lng:         derefF64(t.Longitude),
			WindowStart: buildUnixSec(t.WindowStartDate, t.WindowStartTime),
			WindowEnd:   buildUnixSec(t.WindowEndDate, t.WindowEndTime),
			DurationMin: derefInt(t.DurationMin),
		}
		taskIDToIdx[t.ID] = i
	}

	edges := make([][]routeopt.Edge, len(tasks))
	for i := range tasks {
		edges[i] = make([]routeopt.Edge, len(tasks))
		for j := range tasks {
			edges[i][j] = routeopt.Edge{
				DistanceM:   matrix[i][j].DistanceM,
				DurationSec: matrix[i][j].DurationSec,
			}
		}
	}

	g := &routeopt.Graph{Nodes: nodes, Edges: edges}

	var constraints routeopt.Constraints
	if startTaskID != nil {
		if idx, ok := taskIDToIdx[*startTaskID]; ok {
			constraints.StartNodeIdx = &idx
		}
	}
	if endTaskID != nil {
		if idx, ok := taskIDToIdx[*endTaskID]; ok {
			constraints.EndNodeIdx = &idx
		}
	}
	for _, p := range precedences {
		beforeIdx, bOk := taskIDToIdx[p.BeforeTaskID]
		afterIdx, aOk := taskIDToIdx[p.AfterTaskID]
		if bOk && aOk {
			constraints.PrecedencePairs = append(constraints.PrecedencePairs, routeopt.PrecedencePair{
				Before: beforeIdx,
				After:  afterIdx,
			})
		}
	}

	result, err := s.optimizer.Optimize(ctx, g, startTimeUnix, constraints)
	if err != nil {
		return route.Full{}, fmt.Errorf("optimise: %w", err)
	}

	stops := make([]routegate.StopInput, len(result.Order))
	for pos, nodeIdx := range result.Order {
		travelSec := 0
		if pos > 0 {
			travelSec = matrix[result.Order[pos-1]][nodeIdx].DurationSec
		}
		stops[pos] = routegate.StopInput{
			TaskID:            tasks[nodeIdx].ID,
			Position:          pos,
			TravelFromPrevSec: travelSec,
		}
	}

	// Сохраняем всё в одной транзакции.
	rt, err := s.routes.SaveOptimizedRoute(
		ctx, userID, s.optimizer.Name(), stops,
		result.TotalDistanceM, result.TotalTravelSec,
		result.TotalServiceSec, result.TotalWaitSec,
	)
	if err != nil {
		return route.Full{}, fmt.Errorf("save route: %w", err)
	}

	// Собираем ответ из данных в памяти — лишнего запроса к БД нет.
	routeStops := make([]route.Stop, len(stops))
	for i, stop := range stops {
		travel := stop.TravelFromPrevSec
		routeStops[i] = route.Stop{
			RouteID:           rt.ID,
			TaskID:            stop.TaskID,
			Position:          stop.Position,
			TravelFromPrevSec: &travel,
		}
	}

	distM := result.TotalDistanceM
	travelSec := result.TotalTravelSec
	serviceSec := result.TotalServiceSec
	waitSec := result.TotalWaitSec
	stats := &route.Stats{
		RouteID:         rt.ID,
		TotalDistanceM:  &distM,
		TotalTravelSec:  &travelSec,
		TotalServiceSec: &serviceSec,
		TotalWaitSec:    &waitSec,
	}

	return route.Full{Route: rt, Stops: routeStops, Stats: stats}, nil
}

func (s *Story) List(ctx context.Context, userID string) ([]route.Route, error) {
	return s.routes.ListByUser(ctx, userID)
}

func (s *Story) Get(ctx context.Context, userID, routeID string) (route.Full, bool, error) {
	if routeID == "" {
		return route.Full{}, false, errors.New("route_id required")
	}
	return s.routes.GetFull(ctx, userID, routeID)
}

func (s *Story) Delete(ctx context.Context, userID, routeID string) (bool, error) {
	if routeID == "" {
		return false, errors.New("route_id required")
	}
	return s.routes.DeleteRoute(ctx, userID, routeID)
}

func (s *Story) DeleteAll(ctx context.Context, userID string) error {
	return s.routes.DeleteAllRoutes(ctx, userID)
}

func (s *Story) Rename(ctx context.Context, userID, routeID, name string) (bool, error) {
	if routeID == "" {
		return false, errors.New("route_id required")
	}
	if name == "" {
		return false, errors.New("name required")
	}
	return s.routes.RenameRoute(ctx, userID, routeID, name)
}

// derefInt safely dereferences a *int, returning 0 for nil.
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// derefF64 safely dereferences a *float64, returning 0 for nil.
func derefF64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// buildUnixSec собирает Unix-секунды из раздельных даты ("YYYY-MM-DD") и времени ("HH:MM").
// Если дата nil/пустая — возвращает -1 (нет ограничения).
// Если дата есть, а время нет — возвращает начало дня (00:00) для start или конец дня (23:59) для end.
// Для простоты: без времени возвращаем начало дня (00:00).
func buildUnixSec(dateStr, timeStr *string) int64 {
	if dateStr == nil || *dateStr == "" {
		return -1
	}
	if timeStr == nil || *timeStr == "" {
		// Дата без времени — начало дня
		t, err := time.Parse("2006-01-02", *dateStr)
		if err != nil {
			return -1
		}
		return t.Unix()
	}
	t, err := time.Parse("2006-01-02 15:04", *dateStr+" "+*timeStr)
	if err != nil {
		return -1
	}
	return t.Unix()
}
