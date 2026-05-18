package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"planner-backend/internal/common/ptrs"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
	routeopt "planner-backend/internal/domain/route/optimizer"
	routestory "planner-backend/internal/domain/route/story"
	"planner-backend/internal/domain/task"
	distancepkg "planner-backend/internal/platform/distance"
)

type hRouteRepo struct {
	saveOptimizedRouteFn func(ctx context.Context, userID, algorithm string, stops []routegate.StopInput, distanceM, travelSec, serviceSec, waitSec int) (route.Route, error)
}

func (m *hRouteRepo) SaveOptimizedRoute(ctx context.Context, userID, algorithm string, stops []routegate.StopInput, distanceM, travelSec, serviceSec, waitSec int) (route.Route, error) {
	if m.saveOptimizedRouteFn != nil {
		return m.saveOptimizedRouteFn(ctx, userID, algorithm, stops, distanceM, travelSec, serviceSec, waitSec)
	}
	return route.Route{}, nil
}

type hTaskRepo struct {
	getByIDsFn func(ctx context.Context, userID string, ids []string) ([]task.Task, error)
}

func (m *hTaskRepo) ListByUser(ctx context.Context, _ string) ([]task.Task, error) { return nil, nil }
func (m *hTaskRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, userID, ids)
	}
	return []task.Task{}, nil
}
func (m *hTaskRepo) Create(ctx context.Context, _ string, _ task.CreateInput) (task.Task, error) {
	return task.Task{}, nil
}
func (m *hTaskRepo) Update(ctx context.Context, _, _ string, _ task.UpdateInput) (task.Task, bool, error) {
	return task.Task{}, false, nil
}
func (m *hTaskRepo) Delete(ctx context.Context, _, _ string) (bool, error) { return false, nil }
func (m *hTaskRepo) DeleteAll(ctx context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *hTaskRepo) BulkReorder(ctx context.Context, _ string, _ task.ReorderInput) error {
	return nil
}
func (m *hTaskRepo) BatchCreate(ctx context.Context, _ string, _ []task.CreateInput) ([]task.Task, error) {
	return nil, nil
}

type hDistProvider struct {
	getMatrixFn func(ctx context.Context, points []distancepkg.Point) ([][]distancepkg.Edge, error)
}

func (m *hDistProvider) GetMatrix(ctx context.Context, points []distancepkg.Point) ([][]distancepkg.Edge, error) {
	if m.getMatrixFn != nil {
		return m.getMatrixFn(ctx, points)
	}
	return nil, nil
}

type hOptimizer struct{}

func (o *hOptimizer) Name() string { return "nearest-neighbor-tw" }
func (o *hOptimizer) Optimize(_ context.Context, g *routeopt.Graph, startTime int64, _ routeopt.Constraints) (routeopt.Result, error) {
	order := make([]int, len(g.Nodes))
	timings := make([]routeopt.StopTiming, len(g.Nodes))
	cur := startTime
	for i := range order {
		order[i] = i
		timings[i] = routeopt.StopTiming{NodeIdx: i, ArrivalSec: cur, ServiceStartSec: cur, ServiceEndSec: cur + 600}
		cur += 1200
		if i > 0 {
			timings[i].TravelFromPrevSec = 600
		}
	}
	return routeopt.Result{Order: order, Timings: timings, TotalDistanceM: 1000, TotalTravelSec: 600, TotalServiceSec: 1800}, nil
}

const routeTestUserID = "route-test-user-id"

func newRouteTestRouter(rRepo *hRouteRepo, tRepo *hTaskRepo, dist *hDistProvider) *gin.Engine {
	story := routestory.New(rRepo, tRepo, dist, &hOptimizer{})
	routeH := NewRouteHandlers(story)

	r := gin.New()
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set(ctxUserIDKey, routeTestUserID)
		c.Next()
	})

	api.POST("/routes/optimize", routeH.Optimize)
	return r
}

func makeTestRoute(id, userID string) route.Route {
	return route.Route{
		ID:         id,
		UserID:     userID,
		Algorithm:  "nearest-neighbor-tw",
		ComputedAt: time.Now(),
	}
}

func makeTestTask2(id, userID string) task.Task {
	lat := 55.75
	lon := 37.61
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       "Task " + id,
		AddressText: "Addr",
		Latitude:    &lat,
		Longitude:   &lon,
		DurationMin: ptrs.Ptr(30),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// POST /api/routes/optimize — успех с ≥2 задачами
func TestRouteHandler_Optimize_Success(t *testing.T) {
	routeID := uuid.NewString()
	t1 := makeTestTask2(uuid.NewString(), routeTestUserID)
	t2 := makeTestTask2(uuid.NewString(), routeTestUserID)
	t3 := makeTestTask2(uuid.NewString(), routeTestUserID)

	matrix := [][]DistanceCellDTO{
		{{}, {DistanceM: 1000, DurationSec: 600}, {DistanceM: 2000, DurationSec: 1200}},
		{{DistanceM: 1000, DurationSec: 600}, {}, {DistanceM: 1000, DurationSec: 600}},
		{{DistanceM: 2000, DurationSec: 1200}, {DistanceM: 1000, DurationSec: 600}, {}},
	}

	tRepo := &hTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2, t3}, nil
		},
	}
	rRepo := &hRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeTestRoute(routeID, routeTestUserID), nil
		},
	}

	r := newRouteTestRouter(rRepo, tRepo, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{
		"taskIds":        []string{t1.ID, t2.ID, t3.ID},
		"startTimeUnix":  1704085200,
		"distanceMatrix": matrix,
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// POST /api/routes/optimize — ограничения начала, конца и предшествования
func TestRouteHandler_Optimize_WithConstraints(t *testing.T) {
	routeID := uuid.NewString()
	t1 := makeTestTask2(uuid.NewString(), routeTestUserID)
	t2 := makeTestTask2(uuid.NewString(), routeTestUserID)
	t3 := makeTestTask2(uuid.NewString(), routeTestUserID)

	matrix := [][]DistanceCellDTO{
		{{}, {DistanceM: 1000, DurationSec: 600}, {DistanceM: 2000, DurationSec: 1200}},
		{{DistanceM: 1000, DurationSec: 600}, {}, {DistanceM: 1000, DurationSec: 600}},
		{{DistanceM: 2000, DurationSec: 1200}, {DistanceM: 1000, DurationSec: 600}, {}},
	}

	tRepo := &hTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2, t3}, nil
		},
	}
	rRepo := &hRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeTestRoute(routeID, routeTestUserID), nil
		},
	}

	r := newRouteTestRouter(rRepo, tRepo, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{
		"taskIds":        []string{t1.ID, t2.ID, t3.ID},
		"startTimeUnix":  1704085200,
		"distanceMatrix": matrix,
		"startTaskId":    t1.ID,
		"endTaskId":      t3.ID,
		"precedenceConstraints": []map[string]string{
			{"beforeTaskId": t1.ID, "afterTaskId": t2.ID},
		},
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// POST /api/routes/optimize — 1 задача → 400
func TestRouteHandler_Optimize_OnlyOneTask(t *testing.T) {
	tRepo := &hTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{makeTestTask2(uuid.NewString(), routeTestUserID)}, nil
		},
	}
	r := newRouteTestRouter(&hRouteRepo{}, tRepo, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{
		"taskIds": []string{"single-task"},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// POST /api/routes/optimize — без taskIds → 400
func TestRouteHandler_Optimize_NoTaskIDs(t *testing.T) {
	r := newRouteTestRouter(&hRouteRepo{}, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
