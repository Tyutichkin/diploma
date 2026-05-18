package story

import (
	"context"
	"errors"
	"fmt"
	"time"

	"planner-backend/internal/common/ptrs"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
	routeopt "planner-backend/internal/domain/route/optimizer"
	"planner-backend/internal/domain/task"
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

type PrecedenceConstraint struct {
	BeforeTaskID string
	AfterTaskID  string
}

// externalMatrix[i][j] индексируется по taskIDs (DTO-контракт), а репозиторий может
// вернуть задачи в произвольном порядке — поэтому tasks выравниваются по taskIDs.
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

	fetched, err := s.tasks.GetByIDs(ctx, userID, taskIDs)
	if err != nil {
		return route.Full{}, fmt.Errorf("fetch tasks: %w", err)
	}

	tasks, foundIdxs := alignTasksToIDs(fetched, taskIDs)
	if len(tasks) < 2 {
		return route.Full{}, errors.New("at least 2 valid tasks required")
	}
	if len(externalMatrix) > 0 {
		externalMatrix = subMatrix(externalMatrix, foundIdxs)
	}

	matrix, err := s.resolveMatrix(ctx, tasks, externalMatrix)
	if err != nil {
		return route.Full{}, err
	}

	graph, taskIDToIdx := buildGraph(tasks, matrix, time.Unix(startTimeUnix, 0).UTC())
	startTimeUnix = adjustStartTime(graph.Nodes, startTimeUnix)

	constraints := buildConstraints(taskIDToIdx, startTaskID, endTaskID, precedences)

	result, err := s.optimizer.Optimize(ctx, graph, startTimeUnix, constraints)
	if err != nil {
		return route.Full{}, fmt.Errorf("optimise: %w", err)
	}

	stops := makeStopInputs(result, tasks)

	rt, err := s.routes.SaveOptimizedRoute(
		ctx, userID, s.optimizer.Name(), stops,
		result.TotalDistanceM, result.TotalTravelSec,
		result.TotalServiceSec, result.TotalWaitSec,
	)
	if err != nil {
		return route.Full{}, fmt.Errorf("save route: %w", err)
	}

	return assembleFull(rt, stops, result), nil
}

// Возвращает задачи в порядке taskIDs и индексы найденных в исходном taskIDs.
// Неизвестные id пропускаются; дубли игнорируются после первого вхождения.
func alignTasksToIDs(fetched []task.Task, taskIDs []string) ([]task.Task, []int) {
	byID := make(map[string]task.Task, len(fetched))
	for _, t := range fetched {
		byID[t.ID] = t
	}
	aligned := make([]task.Task, 0, len(fetched))
	indices := make([]int, 0, len(fetched))
	seen := make(map[string]struct{}, len(taskIDs))
	for i, id := range taskIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if t, ok := byID[id]; ok {
			aligned = append(aligned, t)
			indices = append(indices, i)
		}
	}
	return aligned, indices
}

func subMatrix(matrix [][]distancepkg.Edge, indices []int) [][]distancepkg.Edge {
	n := len(indices)
	out := make([][]distancepkg.Edge, n)
	for i, ri := range indices {
		out[i] = make([]distancepkg.Edge, n)
		if ri >= len(matrix) {
			continue
		}
		row := matrix[ri]
		for j, rj := range indices {
			if rj >= len(row) {
				continue
			}
			out[i][j] = row[rj]
		}
	}
	return out
}

func (s *Story) resolveMatrix(
	ctx context.Context,
	tasks []task.Task,
	externalMatrix [][]distancepkg.Edge,
) ([][]distancepkg.Edge, error) {
	if len(externalMatrix) > 0 {
		return externalMatrix, nil
	}

	points := make([]distancepkg.Point, len(tasks))
	for i, t := range tasks {
		if t.Latitude == nil || t.Longitude == nil {
			return nil, fmt.Errorf("task %q has no coordinates: required for distance matrix", t.ID)
		}
		points[i] = distancepkg.Point{Lat: *t.Latitude, Lng: *t.Longitude}
	}

	matrix, err := s.distProvider.GetMatrix(ctx, points)
	if err != nil {
		return nil, fmt.Errorf("distance matrix: %w", err)
	}
	return matrix, nil
}

func buildGraph(
	tasks []task.Task,
	matrix [][]distancepkg.Edge,
	fallbackDate time.Time,
) (*routeopt.Graph, map[string]int) {
	n := len(tasks)
	nodes := make([]routeopt.Node, n)
	idx := make(map[string]int, n)
	for i, t := range tasks {
		nodes[i] = routeopt.Node{
			TaskID:      t.ID,
			Lat:         ptrs.Deref(t.Latitude),
			Lng:         ptrs.Deref(t.Longitude),
			WindowStart: buildUnixSec(t.WindowStartDate, t.WindowStartTime, fallbackDate, boundStart),
			WindowEnd:   buildUnixSec(t.WindowEndDate, t.WindowEndTime, fallbackDate, boundEnd),
			DurationMin: ptrs.Deref(t.DurationMin),
		}
		idx[t.ID] = i
	}

	edges := make([][]routeopt.Edge, n)
	for i := 0; i < n; i++ {
		edges[i] = make([]routeopt.Edge, n)
		for j := 0; j < n; j++ {
			edges[i][j] = routeopt.Edge{
				DistanceM:   matrix[i][j].DistanceM,
				DurationSec: matrix[i][j].DurationSec,
			}
		}
	}

	return &routeopt.Graph{Nodes: nodes, Edges: edges}, idx
}

// Сдвигает старт к самому раннему WindowStart, если он раньше startTimeUnix —
// иначе утренние окна сочлись бы просроченными при startTimeUnix=time.Now().
func adjustStartTime(nodes []routeopt.Node, startTimeUnix int64) int64 {
	for _, nd := range nodes {
		if nd.WindowStart >= 0 && nd.WindowStart < startTimeUnix {
			startTimeUnix = nd.WindowStart
		}
	}
	return startTimeUnix
}

// buildConstraints разрешает идентификаторы задач в индексы узлов графа.
// Неизвестные ID игнорируются (а не приводят к ошибке) — это исторически согласовано
// с фронтом, который может прислать удалённую/чужую задачу.
func buildConstraints(
	taskIDToIdx map[string]int,
	startTaskID, endTaskID *string,
	precedences []PrecedenceConstraint,
) routeopt.Constraints {
	var c routeopt.Constraints
	if startTaskID != nil {
		if i, ok := taskIDToIdx[*startTaskID]; ok {
			c.StartNodeIdx = &i
		}
	}
	if endTaskID != nil {
		if i, ok := taskIDToIdx[*endTaskID]; ok {
			c.EndNodeIdx = &i
		}
	}
	for _, p := range precedences {
		bi, bOk := taskIDToIdx[p.BeforeTaskID]
		ai, aOk := taskIDToIdx[p.AfterTaskID]
		if bOk && aOk {
			c.PrecedencePairs = append(c.PrecedencePairs, routeopt.PrecedencePair{
				Before: bi,
				After:  ai,
			})
		}
	}
	return c
}

func makeStopInputs(result routeopt.Result, tasks []task.Task) []routegate.StopInput {
	stops := make([]routegate.StopInput, len(result.Order))
	for pos, nodeIdx := range result.Order {
		t := result.Timings[pos]
		arrive := time.Unix(t.ArrivalSec, 0).UTC()
		serviceStart := time.Unix(t.ServiceStartSec, 0).UTC()
		serviceEnd := time.Unix(t.ServiceEndSec, 0).UTC()
		wait := t.WaitSec

		stops[pos] = routegate.StopInput{
			TaskID:            tasks[nodeIdx].ID,
			Position:          pos,
			TravelFromPrevSec: t.TravelFromPrevSec,
			ArriveTime:        &arrive,
			ServiceStartTime:  &serviceStart,
			ServiceEndTime:    &serviceEnd,
			WaitSec:           &wait,
		}
	}
	return stops
}

func assembleFull(rt route.Route, stops []routegate.StopInput, _ routeopt.Result) route.Full {
	routeStops := make([]route.Stop, len(stops))
	for i, s := range stops {
		travel := s.TravelFromPrevSec
		routeStops[i] = route.Stop{
			RouteID:           rt.ID,
			TaskID:            s.TaskID,
			Position:          s.Position,
			TravelFromPrevSec: &travel,
			ArriveTime:        s.ArriveTime,
			ServiceStartTime:  s.ServiceStartTime,
			ServiceEndTime:    s.ServiceEndTime,
			WaitSec:           s.WaitSec,
		}
	}

	return route.Full{Route: rt, Stops: routeStops}
}

type boundKind int

const (
	boundStart boundKind = iota
	boundEnd
)

// «Только дата» раскрывается в полный день: 00:00 для начала, 23:59 для конца.
// Возвращает -1, если ни дата, ни время не заданы.
func buildUnixSec(dateStr, timeStr *string, fallbackDate time.Time, kind boundKind) int64 {
	hasDate := dateStr != nil && *dateStr != ""
	hasTime := timeStr != nil && *timeStr != ""

	if !hasDate && !hasTime {
		return -1
	}

	datePart := fallbackDate.Format("2006-01-02")
	if hasDate {
		datePart = *dateStr
	}

	timePart := ""
	if hasTime {
		timePart = *timeStr
	} else if kind == boundEnd {
		timePart = "23:59"
	} else {
		timePart = "00:00"
	}

	t, err := time.Parse("2006-01-02 15:04", datePart+" "+timePart)
	if err != nil {
		return -1
	}
	return t.Unix()
}
