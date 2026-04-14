package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
	routeopt "planner-backend/internal/domain/route/optimizer"
	routestory "planner-backend/internal/domain/route/story"
	"planner-backend/internal/domain/task"
	distancepkg "planner-backend/internal/platform/distance"
)

// ── mock repos for route handler tests ───────────────────────────────────────

type hRouteRepo struct {
	createDraftFn        func(ctx context.Context, userID, source string) (route.Route, error)
	listByUserFn         func(ctx context.Context, userID string) ([]route.Route, error)
	replaceStopsFn       func(ctx context.Context, routeID string, taskIDs []string) error
	getFullFn            func(ctx context.Context, userID, routeID string) (route.Full, bool, error)
	markFailedFn         func(ctx context.Context, routeID string) error
	saveOptimizedRouteFn func(ctx context.Context, userID, algorithm string, stops []routegate.StopInput, distanceM, travelSec, serviceSec, waitSec int) (route.Route, error)
	deleteRouteFn        func(ctx context.Context, userID, routeID string) (bool, error)
	deleteAllRoutesFn    func(ctx context.Context, userID string) error
	renameRouteFn        func(ctx context.Context, userID, routeID, name string) (bool, error)
}

func (m *hRouteRepo) CreateDraft(ctx context.Context, userID, source string) (route.Route, error) {
	if m.createDraftFn != nil {
		return m.createDraftFn(ctx, userID, source)
	}
	return route.Route{}, nil
}
func (m *hRouteRepo) ListByUser(ctx context.Context, userID string) ([]route.Route, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID)
	}
	return []route.Route{}, nil
}
func (m *hRouteRepo) ReplaceStops(ctx context.Context, routeID string, taskIDs []string) error {
	if m.replaceStopsFn != nil {
		return m.replaceStopsFn(ctx, routeID, taskIDs)
	}
	return nil
}
func (m *hRouteRepo) GetFull(ctx context.Context, userID, routeID string) (route.Full, bool, error) {
	if m.getFullFn != nil {
		return m.getFullFn(ctx, userID, routeID)
	}
	return route.Full{}, false, nil
}
func (m *hRouteRepo) MarkFailed(ctx context.Context, routeID string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, routeID)
	}
	return nil
}
func (m *hRouteRepo) SaveOptimizedRoute(ctx context.Context, userID, algorithm string, stops []routegate.StopInput, distanceM, travelSec, serviceSec, waitSec int) (route.Route, error) {
	if m.saveOptimizedRouteFn != nil {
		return m.saveOptimizedRouteFn(ctx, userID, algorithm, stops, distanceM, travelSec, serviceSec, waitSec)
	}
	return route.Route{}, nil
}
func (m *hRouteRepo) DeleteRoute(ctx context.Context, userID, routeID string) (bool, error) {
	if m.deleteRouteFn != nil {
		return m.deleteRouteFn(ctx, userID, routeID)
	}
	return false, nil
}
func (m *hRouteRepo) DeleteAllRoutes(ctx context.Context, userID string) error {
	if m.deleteAllRoutesFn != nil {
		return m.deleteAllRoutesFn(ctx, userID)
	}
	return nil
}
func (m *hRouteRepo) RenameRoute(ctx context.Context, userID, routeID, name string) (bool, error) {
	if m.renameRouteFn != nil {
		return m.renameRouteFn(ctx, userID, routeID, name)
	}
	return false, nil
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
func (m *hTaskRepo) SoftDelete(ctx context.Context, _, _ string) (bool, error) { return false, nil }
func (m *hTaskRepo) BulkReorder(ctx context.Context, _ string, _ task.ReorderInput) error {
	return nil
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
func (o *hOptimizer) Optimize(_ context.Context, g *routeopt.Graph, _ int64, _ routeopt.Constraints) (routeopt.Result, error) {
	order := make([]int, len(g.Nodes))
	for i := range order {
		order[i] = i
	}
	return routeopt.Result{Order: order, TotalDistanceM: 1000, TotalTravelSec: 600, TotalServiceSec: 1800}, nil
}

// ── test router for routes ────────────────────────────────────────────────────

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

	api.POST("/routes", routeH.Create)
	api.POST("/routes/optimize", routeH.Optimize)
	api.GET("/routes", routeH.List)
	api.DELETE("/routes", routeH.DeleteAll)
	api.GET("/routes/:id", routeH.Get)
	api.DELETE("/routes/:id", routeH.Delete)
	api.PATCH("/routes/:id/name", routeH.Rename)
	return r
}

func makeTestRoute(id, userID, status string) route.Route {
	return route.Route{
		ID:        id,
		UserID:    userID,
		Status:    status,
		Source:    "manual",
		CreatedAt: time.Now(),
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
		DurationMin: intPtr(30),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ── Route handler tests ───────────────────────────────────────────────────────

// 5.4.1 POST /api/routes — успех
func TestRouteHandler_Create_Success(t *testing.T) {
	routeID := uuid.NewString()
	rRepo := &hRouteRepo{
		createDraftFn: func(_ context.Context, _, _ string) (route.Route, error) {
			return makeTestRoute(routeID, routeTestUserID, "draft"), nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes", map[string]any{
		"source": "manual", "orderedTaskIds": []string{},
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var got route.Route
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, routeID, got.ID)
}

// 5.4.2 POST /api/routes/optimize — успех с ≥2 задачами
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
			return makeTestRoute(routeID, routeTestUserID, "optimized"), nil
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

// 5.4.2b POST /api/routes/optimize — ограничения начала, конца и предшествования
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
			return makeTestRoute(routeID, routeTestUserID, "optimized"), nil
		},
	}

	r := newRouteTestRouter(rRepo, tRepo, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{
		"taskIds":       []string{t1.ID, t2.ID, t3.ID},
		"startTimeUnix": 1704085200,
		"distanceMatrix": matrix,
		"startTaskId":   t1.ID,
		"endTaskId":     t3.ID,
		"precedenceConstraints": []map[string]string{
			{"beforeTaskId": t1.ID, "afterTaskId": t2.ID},
		},
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// 5.4.3 POST /api/routes/optimize — 1 задача → 400
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

// 5.4.4 POST /api/routes/optimize — без taskIds → 400
func TestRouteHandler_Optimize_NoTaskIDs(t *testing.T) {
	r := newRouteTestRouter(&hRouteRepo{}, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "POST", "/api/routes/optimize", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5.4.5 GET /api/routes — успех
func TestRouteHandler_List_Success(t *testing.T) {
	routes := []route.Route{
		makeTestRoute(uuid.NewString(), routeTestUserID, "draft"),
		makeTestRoute(uuid.NewString(), routeTestUserID, "optimized"),
	}
	rRepo := &hRouteRepo{
		listByUserFn: func(_ context.Context, _ string) ([]route.Route, error) {
			return routes, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	req := httptest.NewRequest("GET", "/api/routes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var got []route.Route
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got, 2)
}

// 5.4.6 GET /api/routes/:id — успех
func TestRouteHandler_Get_Success(t *testing.T) {
	routeID := uuid.NewString()
	full := route.Full{Route: makeTestRoute(routeID, routeTestUserID, "optimized")}
	rRepo := &hRouteRepo{
		getFullFn: func(_ context.Context, _, _ string) (route.Full, bool, error) {
			return full, true, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	req := httptest.NewRequest("GET", "/api/routes/"+routeID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 5.4.7 GET /api/routes/:id — не найдено → 404
func TestRouteHandler_Get_NotFound(t *testing.T) {
	rRepo := &hRouteRepo{
		getFullFn: func(_ context.Context, _, _ string) (route.Full, bool, error) {
			return route.Full{}, false, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	req := httptest.NewRequest("GET", "/api/routes/"+uuid.NewString(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.4.9 DELETE /api/routes/:id — успех
func TestRouteHandler_Delete_Success(t *testing.T) {
	routeID := uuid.NewString()
	rRepo := &hRouteRepo{
		deleteRouteFn: func(_ context.Context, _, id string) (bool, error) {
			assert.Equal(t, routeID, id)
			return true, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "DELETE", "/api/routes/"+routeID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

// 5.4.10 DELETE /api/routes/:id — не найдено → 404
func TestRouteHandler_Delete_NotFound(t *testing.T) {
	rRepo := &hRouteRepo{
		deleteRouteFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "DELETE", "/api/routes/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.4.11 DELETE /api/routes — удаляет все → 200
func TestRouteHandler_DeleteAll_Success(t *testing.T) {
	rRepo := &hRouteRepo{
		deleteAllRoutesFn: func(_ context.Context, _ string) error { return nil },
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "DELETE", "/api/routes", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

// 5.4.12 PATCH /api/routes/:id/name — успех
func TestRouteHandler_Rename_Success(t *testing.T) {
	routeID := uuid.NewString()
	rRepo := &hRouteRepo{
		renameRouteFn: func(_ context.Context, _, id, name string) (bool, error) {
			assert.Equal(t, routeID, id)
			assert.Equal(t, "My Route", name)
			return true, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "PATCH", "/api/routes/"+routeID+"/name", map[string]string{
		"name": "My Route",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

// 5.4.13 PATCH /api/routes/:id/name — не найдено → 404
func TestRouteHandler_Rename_NotFound(t *testing.T) {
	rRepo := &hRouteRepo{
		renameRouteFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	w := doJSON(t, r, "PATCH", "/api/routes/"+uuid.NewString()+"/name", map[string]string{
		"name": "Any Name",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.4.8 GET /api/routes/:id — чужой маршрут → 404
func TestRouteHandler_Get_OtherUser(t *testing.T) {
	rRepo := &hRouteRepo{
		getFullFn: func(_ context.Context, _, _ string) (route.Full, bool, error) {
			return route.Full{}, false, nil // isolation
		},
	}
	r := newRouteTestRouter(rRepo, &hTaskRepo{}, &hDistProvider{})
	req := httptest.NewRequest("GET", "/api/routes/"+uuid.NewString(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
