package story

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planner-backend/internal/common/ptrs"
	"planner-backend/internal/domain/route"
	routegate "planner-backend/internal/domain/route/gate"
	routeopt "planner-backend/internal/domain/route/optimizer"
	"planner-backend/internal/domain/task"
	distancepkg "planner-backend/internal/platform/distance"
)

type mockRouteRepo struct {
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

func (m *mockRouteRepo) CreateDraft(ctx context.Context, userID, source string) (route.Route, error) {
	return m.createDraftFn(ctx, userID, source)
}
func (m *mockRouteRepo) ListByUser(ctx context.Context, userID string) ([]route.Route, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockRouteRepo) ReplaceStops(ctx context.Context, routeID string, taskIDs []string) error {
	return m.replaceStopsFn(ctx, routeID, taskIDs)
}
func (m *mockRouteRepo) GetFull(ctx context.Context, userID, routeID string) (route.Full, bool, error) {
	return m.getFullFn(ctx, userID, routeID)
}
func (m *mockRouteRepo) MarkFailed(ctx context.Context, routeID string) error {
	return m.markFailedFn(ctx, routeID)
}
func (m *mockRouteRepo) SaveOptimizedRoute(ctx context.Context, userID, algorithm string, stops []routegate.StopInput, distanceM, travelSec, serviceSec, waitSec int) (route.Route, error) {
	return m.saveOptimizedRouteFn(ctx, userID, algorithm, stops, distanceM, travelSec, serviceSec, waitSec)
}
func (m *mockRouteRepo) DeleteRoute(ctx context.Context, userID, routeID string) (bool, error) {
	return m.deleteRouteFn(ctx, userID, routeID)
}
func (m *mockRouteRepo) DeleteAllRoutes(ctx context.Context, userID string) error {
	return m.deleteAllRoutesFn(ctx, userID)
}
func (m *mockRouteRepo) RenameRoute(ctx context.Context, userID, routeID, name string) (bool, error) {
	return m.renameRouteFn(ctx, userID, routeID, name)
}

type mockTaskRepo struct {
	listByUserFn  func(ctx context.Context, userID string) ([]task.Task, error)
	getByIDsFn    func(ctx context.Context, userID string, ids []string) ([]task.Task, error)
	createFn      func(ctx context.Context, userID string, in task.CreateInput) (task.Task, error)
	updateFn      func(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error)
	softDeleteFn  func(ctx context.Context, userID, taskID string) (bool, error)
	bulkReorderFn func(ctx context.Context, userID string, in task.ReorderInput) error
}

func (m *mockTaskRepo) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockTaskRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error) {
	return m.getByIDsFn(ctx, userID, ids)
}
func (m *mockTaskRepo) Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error) {
	return m.createFn(ctx, userID, in)
}
func (m *mockTaskRepo) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	return m.updateFn(ctx, userID, taskID, in)
}
func (m *mockTaskRepo) SoftDelete(ctx context.Context, userID, taskID string) (bool, error) {
	return m.softDeleteFn(ctx, userID, taskID)
}
func (m *mockTaskRepo) SoftDeleteAll(ctx context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockTaskRepo) BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return m.bulkReorderFn(ctx, userID, in)
}
func (m *mockTaskRepo) BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	return nil, nil
}

type mockDistProvider struct {
	getMatrixFn func(ctx context.Context, points []distancepkg.Point) ([][]distancepkg.Edge, error)
}

func (m *mockDistProvider) GetMatrix(ctx context.Context, points []distancepkg.Point) ([][]distancepkg.Edge, error) {
	return m.getMatrixFn(ctx, points)
}

type mockOptimizer struct {
	nameFn     func() string
	optimizeFn func(ctx context.Context, g *routeopt.Graph, startTimeUnix int64, c routeopt.Constraints) (routeopt.Result, error)
}

func (m *mockOptimizer) Name() string { return m.nameFn() }
func (m *mockOptimizer) Optimize(ctx context.Context, g *routeopt.Graph, startTimeUnix int64, c routeopt.Constraints) (routeopt.Result, error) {
	return m.optimizeFn(ctx, g, startTimeUnix, c)
}

func makeRouteTask(id, userID string) task.Task {
	lat := 55.75
	lon := 37.61
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       "Task " + id,
		AddressText: "Some Address",
		Latitude:    &lat,
		Longitude:   &lon,
		DurationMin: ptrs.Ptr(30),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func identity2x2Matrix() [][]distancepkg.Edge {
	return [][]distancepkg.Edge{
		{{DistanceM: 0, DurationSec: 0}, {DistanceM: 1000, DurationSec: 600}},
		{{DistanceM: 1000, DurationSec: 600}, {DistanceM: 0, DurationSec: 0}},
	}
}

func makeRoute(id, userID, status string) route.Route {
	return route.Route{
		ID:        id,
		UserID:    userID,
		Status:    status,
		Source:    "manual",
		CreatedAt: time.Now(),
	}
}

func defaultOptimizer() *mockOptimizer {
	return &mockOptimizer{
		nameFn: func() string { return "nearest-neighbor-tw" },
		optimizeFn: func(_ context.Context, g *routeopt.Graph, startTime int64, _ routeopt.Constraints) (routeopt.Result, error) {
			order := make([]int, len(g.Nodes))
			timings := make([]routeopt.StopTiming, len(g.Nodes))
			cur := startTime
			for i := range order {
				order[i] = i
				timings[i] = routeopt.StopTiming{
					NodeIdx:         i,
					ArrivalSec:      cur,
					ServiceStartSec: cur,
					ServiceEndSec:   cur + int64(g.Nodes[i].DurationMin)*60,
				}
				cur = timings[i].ServiceEndSec + 600 // 10 min travel
				if i > 0 {
					timings[i].TravelFromPrevSec = 600
				}
			}
			return routeopt.Result{
				Order:           order,
				Timings:         timings,
				TotalDistanceM:  1000,
				TotalTravelSec:  600,
				TotalServiceSec: 1800,
			}, nil
		},
	}
}

// 3.1.1 Создание черновика с задачами
func TestCreateDraft_WithTasks(t *testing.T) {
	routeID := uuid.NewString()
	uid := uuid.NewString()
	taskIDs := []string{uuid.NewString(), uuid.NewString()}
	createdRoute := makeRoute(routeID, uid, "draft")

	rRepo := &mockRouteRepo{
		createDraftFn: func(_ context.Context, _, _ string) (route.Route, error) {
			return createdRoute, nil
		},
		replaceStopsFn: func(_ context.Context, rID string, ids []string) error {
			assert.Equal(t, routeID, rID)
			assert.Equal(t, taskIDs, ids)
			return nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	rt, err := s.CreateDraft(context.Background(), uid, "manual", taskIDs)
	require.NoError(t, err)
	assert.Equal(t, routeID, rt.ID)
	assert.Equal(t, "draft", rt.Status)
}

// 3.1.2 Пустой список задач
func TestCreateDraft_EmptyTaskList(t *testing.T) {
	uid := uuid.NewString()
	createdRoute := makeRoute(uuid.NewString(), uid, "draft")

	replaceStopsCalled := false
	rRepo := &mockRouteRepo{
		createDraftFn: func(_ context.Context, _, _ string) (route.Route, error) {
			return createdRoute, nil
		},
		replaceStopsFn: func(_ context.Context, _ string, _ []string) error {
			replaceStopsCalled = true
			return nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	rt, err := s.CreateDraft(context.Background(), uid, "manual", []string{})
	require.NoError(t, err)
	assert.NotEmpty(t, rt.ID)
	assert.False(t, replaceStopsCalled, "ReplaceStops should not be called for empty task list")
}

// 3.2.1 Успешная оптимизация
func TestOptimize_Success(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)
	t3 := makeRouteTask(uuid.NewString(), uid)
	taskIDs := []string{t1.ID, t2.ID, t3.ID}

	matrix := [][]distancepkg.Edge{
		{{}, {DistanceM: 1000, DurationSec: 600}, {DistanceM: 2000, DurationSec: 1200}},
		{{DistanceM: 1000, DurationSec: 600}, {}, {DistanceM: 1000, DurationSec: 600}},
		{{DistanceM: 2000, DurationSec: 1200}, {DistanceM: 1000, DurationSec: 600}, {}},
	}

	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _ string, _ string, stops []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2, t3}, nil
		},
	}

	s := New(rRepo, tRepo, &mockDistProvider{}, defaultOptimizer())
	full, err := s.Optimize(context.Background(), uid, taskIDs, 540, matrix, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "optimized", full.Route.Status)
	assert.NotNil(t, full.Stats)
	assert.Len(t, full.Stops, 3)
}

// 3.2.2 Менее 2 задач
func TestOptimize_LessThanTwoTasks(t *testing.T) {
	s := New(&mockRouteRepo{}, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, err := s.Optimize(context.Background(), "uid", []string{"one-task"}, 540, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2")
}

// 3.2.3 Ровно 2 задачи — успех
func TestOptimize_ExactlyTwoTasks(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)
	taskIDs := []string{t1.ID, t2.ID}
	matrix := identity2x2Matrix()

	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2}, nil
		},
	}

	s := New(rRepo, tRepo, &mockDistProvider{}, defaultOptimizer())
	full, err := s.Optimize(context.Background(), uid, taskIDs, 540, matrix, nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, full.Stops, 2)
}

// 3.2.4 Несуществующие taskIDs (repo возвращает < 2 задач)
func TestOptimize_InvalidTaskIDs(t *testing.T) {
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{}, nil // no tasks found
		},
	}
	s := New(&mockRouteRepo{}, tRepo, &mockDistProvider{}, defaultOptimizer())
	_, err := s.Optimize(context.Background(), "uid", []string{"bad-id-1", "bad-id-2"}, 540, nil, nil, nil, nil)
	assert.Error(t, err)
}

// 3.2.5 Внешняя матрица предоставлена — OSRM не вызывается
func TestOptimize_ExternalMatrix_OSRMNotCalled(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)

	osrmCalled := false
	dist := &mockDistProvider{
		getMatrixFn: func(_ context.Context, _ []distancepkg.Point) ([][]distancepkg.Edge, error) {
			osrmCalled = true
			return nil, nil
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2}, nil
		},
	}
	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}

	s := New(rRepo, tRepo, dist, defaultOptimizer())
	_, err := s.Optimize(context.Background(), uid, []string{t1.ID, t2.ID}, 540, identity2x2Matrix(), nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, osrmCalled, "OSRM should not be called when external matrix is provided")
}

// 3.2.6 Матрица не предоставлена — вызывается OSRM провайдер
func TestOptimize_NoMatrix_OSRMCalled(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)

	osrmCalled := false
	dist := &mockDistProvider{
		getMatrixFn: func(_ context.Context, _ []distancepkg.Point) ([][]distancepkg.Edge, error) {
			osrmCalled = true
			return identity2x2Matrix(), nil
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2}, nil
		},
	}
	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}

	s := New(rRepo, tRepo, dist, defaultOptimizer())
	_, err := s.Optimize(context.Background(), uid, []string{t1.ID, t2.ID}, 540, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, osrmCalled, "OSRM should be called when no external matrix is provided")
}

// 3.2.9 Ошибка OSRM
func TestOptimize_OSRMError(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)

	dist := &mockDistProvider{
		getMatrixFn: func(_ context.Context, _ []distancepkg.Point) ([][]distancepkg.Edge, error) {
			return nil, errors.New("OSRM unavailable")
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2}, nil
		},
	}

	s := New(&mockRouteRepo{}, tRepo, dist, defaultOptimizer())
	_, err := s.Optimize(context.Background(), uid, []string{t1.ID, t2.ID}, 540, nil, nil, nil, nil)
	assert.Error(t, err)
}

// 3.2.7 Ограничения передаются алгоритму
func TestOptimize_ConstraintsPassedToOptimizer(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)
	taskIDs := []string{t1.ID, t2.ID}
	matrix := identity2x2Matrix()

	var capturedConstraints routeopt.Constraints
	opt := &mockOptimizer{
		nameFn: func() string { return "nearest-neighbor-tw" },
		optimizeFn: func(_ context.Context, g *routeopt.Graph, startTime int64, c routeopt.Constraints) (routeopt.Result, error) {
			capturedConstraints = c
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
			return routeopt.Result{Order: order, Timings: timings, TotalDistanceM: 1000, TotalTravelSec: 600}, nil
		},
	}

	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t1, t2}, nil
		},
	}

	startID := t1.ID
	endID := t2.ID
	s := New(rRepo, tRepo, &mockDistProvider{}, opt)
	_, err := s.Optimize(context.Background(), uid, taskIDs, 540, matrix,
		&startID, &endID,
		[]PrecedenceConstraint{{BeforeTaskID: t1.ID, AfterTaskID: t2.ID}},
	)
	require.NoError(t, err)
	require.NotNil(t, capturedConstraints.StartNodeIdx)
	assert.Equal(t, 0, *capturedConstraints.StartNodeIdx)
	require.NotNil(t, capturedConstraints.EndNodeIdx)
	assert.Equal(t, 1, *capturedConstraints.EndNodeIdx)
	require.Len(t, capturedConstraints.PrecedencePairs, 1)
	assert.Equal(t, routeopt.PrecedencePair{Before: 0, After: 1}, capturedConstraints.PrecedencePairs[0])
}

// 3.2.8 Одинаковые start и end task → ошибка
func TestOptimize_SameStartAndEnd(t *testing.T) {
	s := New(&mockRouteRepo{}, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	id := uuid.NewString()
	_, err := s.Optimize(context.Background(), "uid", []string{id, uuid.NewString()}, 540, nil, &id, &id, nil)
	assert.Error(t, err)
}

// 3.3.1 List возвращает только маршруты пользователя
func TestList_ReturnsUserRoutes(t *testing.T) {
	uid := uuid.NewString()
	routes := []route.Route{
		makeRoute(uuid.NewString(), uid, "draft"),
		makeRoute(uuid.NewString(), uid, "optimized"),
	}
	rRepo := &mockRouteRepo{
		listByUserFn: func(_ context.Context, id string) ([]route.Route, error) {
			assert.Equal(t, uid, id)
			return routes, nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	got, err := s.List(context.Background(), uid)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

// 3.3.2 Get несуществующего маршрута
func TestGet_NotFound(t *testing.T) {
	rRepo := &mockRouteRepo{
		getFullFn: func(_ context.Context, _, _ string) (route.Full, bool, error) {
			return route.Full{}, false, nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, found, err := s.Get(context.Background(), "uid", uuid.NewString())
	require.NoError(t, err)
	assert.False(t, found)
}

// 3.3.3 Get чужого маршрута
func TestGet_OtherUsersRoute(t *testing.T) {
	rRepo := &mockRouteRepo{
		getFullFn: func(_ context.Context, _, _ string) (route.Full, bool, error) {
			return route.Full{}, false, nil // isolation
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, found, err := s.Get(context.Background(), "user-a", "route-of-user-b")
	require.NoError(t, err)
	assert.False(t, found)
}

// 3.3.4 Delete своего маршрута
func TestDelete_OwnRoute(t *testing.T) {
	rRepo := &mockRouteRepo{
		deleteRouteFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	found, err := s.Delete(context.Background(), "uid", uuid.NewString())
	require.NoError(t, err)
	assert.True(t, found)
}

// 3.3.5 Delete чужого маршрута
func TestDelete_OtherUsersRoute(t *testing.T) {
	rRepo := &mockRouteRepo{
		deleteRouteFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil // isolation
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	found, err := s.Delete(context.Background(), "user-a", "route-of-user-b")
	require.NoError(t, err)
	assert.False(t, found)
}

// 3.3.6 DeleteAll
func TestDeleteAll_Success(t *testing.T) {
	uid := uuid.NewString()
	rRepo := &mockRouteRepo{
		deleteAllRoutesFn: func(_ context.Context, id string) error {
			assert.Equal(t, uid, id)
			return nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	err := s.DeleteAll(context.Background(), uid)
	assert.NoError(t, err)
}

// 3.3.7 Rename существующего маршрута
func TestRename_Success(t *testing.T) {
	uid := uuid.NewString()
	routeID := uuid.NewString()
	newName := "My Route"

	rRepo := &mockRouteRepo{
		renameRouteFn: func(_ context.Context, _, rID, name string) (bool, error) {
			assert.Equal(t, routeID, rID)
			assert.Equal(t, newName, name)
			return true, nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	found, err := s.Rename(context.Background(), uid, routeID, newName)
	require.NoError(t, err)
	assert.True(t, found)
}

// 3.3.8 Rename несуществующего маршрута
func TestRename_NotFound(t *testing.T) {
	rRepo := &mockRouteRepo{
		renameRouteFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	s := New(rRepo, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	found, err := s.Rename(context.Background(), "uid", uuid.NewString(), "name")
	require.NoError(t, err)
	assert.False(t, found)
}

// 3.3.9 Rename пустой строкой → ошибка
func TestRename_EmptyName(t *testing.T) {
	s := New(&mockRouteRepo{}, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, err := s.Rename(context.Background(), "uid", uuid.NewString(), "")
	assert.Error(t, err)
}

// 3.3.2b Get с пустым routeID → ошибка
func TestGet_EmptyRouteID(t *testing.T) {
	s := New(&mockRouteRepo{}, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, _, err := s.Get(context.Background(), "uid", "")
	assert.Error(t, err)
}

// 3.3.4b Delete с пустым routeID → ошибка
func TestDelete_EmptyRouteID(t *testing.T) {
	s := New(&mockRouteRepo{}, &mockTaskRepo{}, &mockDistProvider{}, defaultOptimizer())
	_, err := s.Delete(context.Background(), "uid", "")
	assert.Error(t, err)
}

var fallbackDate = time.Date(2024, 6, 10, 9, 0, 0, 0, time.UTC)

// 4.3.1 Дата + время
func TestBuildUnixSec_DateAndTime(t *testing.T) {
	d := "2024-01-15"
	tm := "09:30"
	expected := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, buildUnixSec(&d, &tm, fallbackDate, boundStart))
}

// 4.3.2 Только дата без времени, граница "start" — начало дня (00:00)
func TestBuildUnixSec_DateOnly_Start(t *testing.T) {
	d := "2024-01-15"
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, buildUnixSec(&d, nil, fallbackDate, boundStart))
}

// 4.3.2b Только дата без времени, граница "end" — конец дня (23:59).
// Без этого окно из одной даты схлопывается в точку и оптимизатор считает,
// что уложиться невозможно (баг при импорте задач без временных окон).
func TestBuildUnixSec_DateOnly_End(t *testing.T) {
	d := "2024-01-15"
	expected := time.Date(2024, 1, 15, 23, 59, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, buildUnixSec(&d, nil, fallbackDate, boundEnd))
}

// Окно "только дата" должно давать ненулевой интервал (00:00–23:59),
// иначе две задачи с одинаковой датой и без времён помечаются как невыполнимые.
func TestBuildGraph_DateOnlyWindow_FullDay(t *testing.T) {
	d := "2024-01-15"
	tasks := []task.Task{
		{
			ID: "a", Latitude: ptrs.Ptr(0.0), Longitude: ptrs.Ptr(0.0),
			DurationMin:     ptrs.Ptr(20),
			WindowStartDate: &d, WindowEndDate: &d,
		},
		{
			ID: "b", Latitude: ptrs.Ptr(0.0), Longitude: ptrs.Ptr(0.0),
			DurationMin:     ptrs.Ptr(20),
			WindowStartDate: &d, WindowEndDate: &d,
		},
	}
	matrix := [][]distancepkg.Edge{
		{{DistanceM: 0, DurationSec: 0}, {DistanceM: 100, DurationSec: 60}},
		{{DistanceM: 100, DurationSec: 60}, {DistanceM: 0, DurationSec: 0}},
	}
	graph, _ := buildGraph(tasks, matrix, fallbackDate)
	for _, n := range graph.Nodes {
		assert.Less(t, n.WindowStart, n.WindowEnd, "окно должно быть положительной длительности")
		assert.Equal(t, int64(86340), n.WindowEnd-n.WindowStart, "ровно полный день минус минута")
	}
}

// 4.3.3 Время без даты → используется fallbackDate
func TestBuildUnixSec_TimeOnly_UsesFallback(t *testing.T) {
	empty := ""
	tm := "09:30"
	// fallbackDate = 2024-06-10 → результат: 2024-06-10 09:30 UTC
	expected := time.Date(2024, 6, 10, 9, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, buildUnixSec(&empty, &tm, fallbackDate, boundStart))
}

// 4.3.4 nil дата + nil время → -1
func TestBuildUnixSec_NilDate(t *testing.T) {
	assert.Equal(t, int64(-1), buildUnixSec(nil, nil, fallbackDate, boundStart))
}

// 4.3.5 Некорректный формат даты
func TestBuildUnixSec_InvalidDate(t *testing.T) {
	d := "not-a-date"
	assert.Equal(t, int64(-1), buildUnixSec(&d, nil, fallbackDate, boundStart))
}

// 4.3.6 nil дата + время → fallback
func TestBuildUnixSec_NilDateWithTime(t *testing.T) {
	tm := "14:00"
	expected := time.Date(2024, 6, 10, 14, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, buildUnixSec(nil, &tm, fallbackDate, boundStart))
}

// 3.2.10 Регрессия: репозиторий вернул задачи в произвольном порядке,
// внешняя матрица проиндексирована по taskIDs (DTO-контракт). Story должна
// выровнять tasks в порядок taskIDs и переставить матрицу, иначе рёбра графа
// привяжутся не к тем узлам.
//
// Воспроизводит баг "после двух нажатий «Оптимизировать маршрут» ПитерСервис
// вылетает за окно": после первой оптимизации фронт шлёт taskIDs в новом
// порядке, БД возвращает строки в физическом порядке, и матрица перекашивается
// относительно узлов.
func TestOptimize_AlignsMatrixWithTaskIDsOrder(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeRouteTask(uuid.NewString(), uid)
	t2 := makeRouteTask(uuid.NewString(), uid)
	t3 := makeRouteTask(uuid.NewString(), uid)

	// Фронт шлёт задачи в порядке [t1, t2, t3].
	taskIDs := []string{t1.ID, t2.ID, t3.ID}

	// Матрица соответствует taskIDs: t1↔t2 дешёвые рёбра, t3 — далеко.
	matrix := [][]distancepkg.Edge{
		{{}, {DistanceM: 100, DurationSec: 60}, {DistanceM: 9000, DurationSec: 5400}},
		{{DistanceM: 100, DurationSec: 60}, {}, {DistanceM: 9000, DurationSec: 5400}},
		{{DistanceM: 9000, DurationSec: 5400}, {DistanceM: 9000, DurationSec: 5400}, {}},
	}

	// Репозиторий возвращает задачи в РАЗНОМ порядке: [t3, t1, t2] —
	// это поведение Postgres `WHERE id IN (...)` без ORDER BY.
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return []task.Task{t3, t1, t2}, nil
		},
	}

	var capturedGraph *routeopt.Graph
	opt := &mockOptimizer{
		nameFn: func() string { return "nearest-neighbor-tw" },
		optimizeFn: func(_ context.Context, g *routeopt.Graph, startTime int64, _ routeopt.Constraints) (routeopt.Result, error) {
			capturedGraph = g
			order := make([]int, len(g.Nodes))
			timings := make([]routeopt.StopTiming, len(g.Nodes))
			cur := startTime
			for i := range order {
				order[i] = i
				timings[i] = routeopt.StopTiming{NodeIdx: i, ArrivalSec: cur, ServiceStartSec: cur, ServiceEndSec: cur + 600}
				cur = timings[i].ServiceEndSec + int64(g.Edges[i][i].DurationSec)
			}
			return routeopt.Result{Order: order, Timings: timings}, nil
		},
	}
	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}

	s := New(rRepo, tRepo, &mockDistProvider{}, opt)
	_, err := s.Optimize(context.Background(), uid, taskIDs, 540, matrix, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, capturedGraph)
	require.Len(t, capturedGraph.Nodes, 3)

	// Порядок узлов графа должен совпадать с порядком taskIDs.
	assert.Equal(t, t1.ID, capturedGraph.Nodes[0].TaskID)
	assert.Equal(t, t2.ID, capturedGraph.Nodes[1].TaskID)
	assert.Equal(t, t3.ID, capturedGraph.Nodes[2].TaskID)

	// Рёбра графа должны соответствовать тем же индексам, что и матрица.
	// Конкретно: ребро t1→t2 (узлы 0→1) — короткое, t1→t3 (0→2) — длинное.
	assert.Equal(t, 60, capturedGraph.Edges[0][1].DurationSec, "t1→t2 должно быть быстрым")
	assert.Equal(t, 5400, capturedGraph.Edges[0][2].DurationSec, "t1→t3 должно быть медленным")
	assert.Equal(t, 5400, capturedGraph.Edges[1][2].DurationSec, "t2→t3 должно быть медленным")
}

// 3.2.11 Регрессия e2e на реальном оптимизаторе: демо-сценарий из CSV
// (СевЗап, Захаров, Фармакор, ПитерСервис, Новикова, Склад, ИнтерТек). Перед
// фиксом в story.go при произвольном порядке возврата из репозитория ПитерСервис
// (окно 11:30–13:30) ставился последним и вылетал за дедлайн.
//
// Сценарий пользователя: после первой оптимизации фронт сохраняет новый порядок
// в БД и шлёт taskIDs в этом порядке на втором клике, а Postgres `WHERE id IN (...)`
// без ORDER BY возвращает строки в физическом heap-порядке — рассогласование
// с матрицей. Здесь это моделируется reverse-shuffle ответа репозитория.
func TestOptimize_DemoCSV_PitersServisFitsWindow(t *testing.T) {
	uid := uuid.NewString()
	day := "2026-05-06"
	mkWindowed := func(title, ws, we string, dur int) task.Task {
		lat, lon := 59.93, 30.36 // координаты не важны: матрица передаётся явно
		wsCopy, weCopy, dayCopy := ws, we, day
		return task.Task{
			ID:              uuid.NewString(),
			UserID:          uid,
			Title:           title,
			AddressText:     title,
			Latitude:        &lat,
			Longitude:       &lon,
			DurationMin:     ptrs.Ptr(dur),
			WindowStartDate: &dayCopy,
			WindowStartTime: &wsCopy,
			WindowEndDate:   &dayCopy,
			WindowEndTime:   &weCopy,
		}
	}
	mkNoWindow := func(title string, dur int) task.Task {
		lat, lon := 59.93, 30.36
		return task.Task{
			ID:          uuid.NewString(),
			UserID:      uid,
			Title:       title,
			AddressText: title,
			Latitude:    &lat,
			Longitude:   &lon,
			DurationMin: ptrs.Ptr(dur),
		}
	}

	// Порядок CSV: СевЗап, Захаров, Фармакор, ПитерСервис, Новикова, Склад, ИнтерТек.
	tasks := []task.Task{
		mkWindowed("СевЗап Логистик", "09:30", "11:00", 30),       // 0
		mkNoWindow("Клиент Захаров", 20),                           // 1
		mkWindowed("Аптечная сеть Фармакор", "10:00", "12:00", 15), // 2
		mkWindowed("ООО ПитерСервис", "11:30", "13:30", 45),        // 3
		mkWindowed("Клиент Новикова", "13:00", "15:00", 30),        // 4
		mkNoWindow("Склад №2", 20),                                 // 5
		mkWindowed("ЗАО ИнтерТек", "15:30", "17:30", 40),           // 6
	}
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}

	// Матрица собрана так, чтобы единственный путь, при котором ПитерСервис
	// (idx 3) укладывается в окно 11:30–13:30, проходил через Фармакор (idx 2):
	// matrix[2][3] = 60 сек (короткий «эксклюзивный» переход), все остальные
	// рёбра, ведущие к ПитерСервису, — 4 часа. При корректном выравнивании
	// matrix↔tasks алгоритм находит этот путь. При перепутанных индексах
	// «эксклюзивная» 60-секундная ячейка применяется к чужой паре узлов, и
	// ПитерСервис становится недостижимым в окне → попадает в pickNextFallback
	// и приезд оказывается после дедлайна.
	n := len(tasks)
	const fast = 600
	const slow = 14400
	matrix := make([][]distancepkg.Edge, n)
	for i := range matrix {
		matrix[i] = make([]distancepkg.Edge, n)
		for j := range matrix[i] {
			if i == j {
				continue
			}
			matrix[i][j] = distancepkg.Edge{DistanceM: 5000, DurationSec: fast}
		}
	}
	// Все рёбра «куда-то к ПитерСервису» — медленные, кроме Фармакор→ПитерСервис.
	for i := 0; i < n; i++ {
		if i == 3 {
			continue
		}
		matrix[i][3] = distancepkg.Edge{DistanceM: 50000, DurationSec: slow}
	}
	matrix[2][3] = distancepkg.Edge{DistanceM: 200, DurationSec: 60}

	// Репозиторий возвращает задачи в обратном порядке — самый агрессивный кейс
	// несоответствия порядка taskIDs и физического порядка строк в БД.
	shuffled := make([]task.Task, n)
	for i, t := range tasks {
		shuffled[n-1-i] = t
	}
	tRepo := &mockTaskRepo{
		getByIDsFn: func(_ context.Context, _ string, _ []string) ([]task.Task, error) {
			return shuffled, nil
		},
	}

	rRepo := &mockRouteRepo{
		saveOptimizedRouteFn: func(_ context.Context, _, _ string, _ []routegate.StopInput, _, _, _, _ int) (route.Route, error) {
			return makeRoute(uuid.NewString(), uid, "optimized"), nil
		},
	}

	// startTimeUnix = 09:30 этого дня — типичное начало рабочего дня; алгоритм
	// внутри сдвигает старт к самому раннему окну, если оно ещё раньше.
	startTimeUnix := time.Date(2026, 5, 6, 9, 30, 0, 0, time.UTC).Unix()

	realOpt := routeopt.NewNearestNeighborTW()
	s := New(rRepo, tRepo, &mockDistProvider{}, realOpt)

	full, err := s.Optimize(context.Background(), uid, taskIDs, startTimeUnix, matrix, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, full.Stops, n)

	// Находим стоп ПитерСервиса и проверяем, что он укладывается в окно 11:30–13:30
	// (start_service ≤ window_end - duration). Без выравнивания матрицы оптимизатор
	// сваливался в pickNextFallback и приезд оказывался после 17:00.
	piterID := tasks[3].ID
	piterEnd := time.Date(2026, 5, 6, 13, 30, 0, 0, time.UTC).Unix()
	piterDur := int64(45 * 60)

	var piterStop *route.Stop
	for i := range full.Stops {
		if full.Stops[i].TaskID == piterID {
			piterStop = &full.Stops[i]
			break
		}
	}
	require.NotNil(t, piterStop, "ПитерСервис должен быть в маршруте")
	require.NotNil(t, piterStop.ServiceStartTime)
	startUnix := piterStop.ServiceStartTime.Unix()
	assert.LessOrEqual(t, startUnix, piterEnd-piterDur,
		"ПитерСервис должен начинаться не позже 12:45 (окно 11:30–13:30 минус 45 мин)")
}
