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

	"planner-backend/internal/common/ptrs"
	"planner-backend/internal/domain/task"
	taskstory "planner-backend/internal/domain/task/story"
)

type handlerTaskRepo struct {
	listByUserFn  func(ctx context.Context, userID string) ([]task.Task, error)
	getByIDsFn    func(ctx context.Context, userID string, ids []string) ([]task.Task, error)
	createFn      func(ctx context.Context, userID string, in task.CreateInput) (task.Task, error)
	updateFn      func(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error)
	softDeleteFn  func(ctx context.Context, userID, taskID string) (bool, error)
	bulkReorderFn func(ctx context.Context, userID string, in task.ReorderInput) error
}

func (m *handlerTaskRepo) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID)
	}
	return []task.Task{}, nil
}
func (m *handlerTaskRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]task.Task, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, userID, ids)
	}
	return []task.Task{}, nil
}
func (m *handlerTaskRepo) Create(ctx context.Context, userID string, in task.CreateInput) (task.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, in)
	}
	return task.Task{}, nil
}
func (m *handlerTaskRepo) Update(ctx context.Context, userID, taskID string, in task.UpdateInput) (task.Task, bool, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, userID, taskID, in)
	}
	return task.Task{}, false, nil
}
func (m *handlerTaskRepo) SoftDelete(ctx context.Context, userID, taskID string) (bool, error) {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, userID, taskID)
	}
	return false, nil
}
func (m *handlerTaskRepo) BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error {
	if m.bulkReorderFn != nil {
		return m.bulkReorderFn(ctx, userID, in)
	}
	return nil
}
func (m *handlerTaskRepo) BatchCreate(ctx context.Context, _ string, _ []task.CreateInput) ([]task.Task, error) {
	return nil, nil
}

const taskTestUserID = "test-user-id"

// newTaskTestRouter собирает роутер с подставленным userID вместо JWT-middleware.
func newTaskTestRouter(repo *handlerTaskRepo) *gin.Engine {
	story := taskstory.New(repo)
	taskH := NewTaskHandlers(story)

	r := gin.New()
	api := r.Group("/api")

	api.Use(func(c *gin.Context) {
		c.Set(ctxUserIDKey, taskTestUserID)
		c.Next()
	})

	api.GET("/tasks", taskH.List)
	api.POST("/tasks", taskH.Create)
	api.PATCH("/tasks/order", taskH.Reorder)
	api.PATCH("/tasks/:id", taskH.Update)
	api.DELETE("/tasks/:id", taskH.Delete)
	return r
}

func makeTestTask(id, userID, title string) task.Task {
	lat := 55.75
	lon := 37.61
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		AddressText: "Test Address",
		Latitude:    &lat,
		Longitude:   &lon,
		DurationMin: ptrs.Ptr(30),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func makeTestTaskNoAddr(id, userID, title string) task.Task {
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		AddressText: "",
		Latitude:    nil,
		Longitude:   nil,
		DurationMin: ptrs.Ptr(15),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// 5.3.1 GET /api/tasks — успех
func TestTaskHandler_List_Success(t *testing.T) {
	tasks := []task.Task{
		makeTestTask(uuid.NewString(), taskTestUserID, "Task A"),
		makeTestTask(uuid.NewString(), taskTestUserID, "Task B"),
	}
	repo := &handlerTaskRepo{
		listByUserFn: func(_ context.Context, _ string) ([]task.Task, error) {
			return tasks, nil
		},
	}
	r := newTaskTestRouter(repo)
	req := httptest.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var got []task.Task
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got, 2)
}

// 5.3.2 GET /api/tasks — без авторизации → 401
func TestTaskHandler_List_Unauthorized(t *testing.T) {
	repo := &handlerTaskRepo{}
	story := taskstory.New(repo)
	taskH := NewTaskHandlers(story)
	r := gin.New()
	r.GET("/api/tasks", taskH.List) // no auth middleware
	req := httptest.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.3.3 POST /api/tasks — успех
func TestTaskHandler_Create_Success(t *testing.T) {
	taskID := uuid.NewString()
	repo := &handlerTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			return makeTestTask(taskID, taskTestUserID, in.Title), nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "POST", "/api/tasks", map[string]any{
		"title": "New Task", "addressText": "123 Main St",
		"latitude": 55.75, "longitude": 37.61, "durationMin": 30,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var got task.Task
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, taskID, got.ID)
}

// 5.3.3b POST /api/tasks — задача без адреса → 200
func TestTaskHandler_Create_NoAddress(t *testing.T) {
	taskID := uuid.NewString()
	repo := &handlerTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			assert.Equal(t, "", in.AddressText)
			assert.Nil(t, in.Latitude)
			assert.Nil(t, in.Longitude)
			return makeTestTaskNoAddr(taskID, taskTestUserID, in.Title), nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "POST", "/api/tasks", map[string]any{
		"title": "Звонок клиенту", "durationMin": 15,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var got task.Task
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, taskID, got.ID)
	assert.Nil(t, got.Latitude)
	assert.Nil(t, got.Longitude)
}

// 5.3.3c POST /api/tasks — явный null для addressText → 200 (фронтенд отправляет null)
func TestTaskHandler_Create_NullAddressText(t *testing.T) {
	taskID := uuid.NewString()
	repo := &handlerTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			assert.Equal(t, "", in.AddressText)
			return makeTestTaskNoAddr(taskID, taskTestUserID, in.Title), nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "POST", "/api/tasks", map[string]any{
		"title": "Звонок", "addressText": nil, "durationMin": 15,
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// 5.3.4 POST /api/tasks — пустое тело → 400
func TestTaskHandler_Create_EmptyTitle(t *testing.T) {
	repo := &handlerTaskRepo{}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "POST", "/api/tasks", map[string]any{
		"title": "", "addressText": "addr", "durationMin": 30,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5.3.5 POST /api/tasks — некорректный JSON → 400
func TestTaskHandler_Create_BrokenJSON(t *testing.T) {
	r := newTaskTestRouter(&handlerTaskRepo{})
	req := httptest.NewRequest("POST", "/api/tasks", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// 5.3.7 PATCH /api/tasks/:id — успех
func TestTaskHandler_Update_Success(t *testing.T) {
	taskID := uuid.NewString()
	newTitle := "Updated Title"
	repo := &handlerTaskRepo{
		updateFn: func(_ context.Context, _, id string, in task.UpdateInput) (task.Task, bool, error) {
			assert.Equal(t, taskID, id)
			t := makeTestTask(taskID, taskTestUserID, *in.Title)
			return t, true, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/"+taskID, map[string]any{
		"title": newTitle,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var got task.Task
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, newTitle, got.Title)
}

// 5.3.7a PATCH /api/tasks/:id — сброс начала окна (windowStart="") → 200, WindowStart nil в repo
func TestTaskHandler_Update_ClearWindowStart(t *testing.T) {
	taskID := uuid.NewString()
	var gotInput task.UpdateInput
	repo := &handlerTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			gotInput = in
			return makeTestTask(taskID, taskTestUserID, "task"), true, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/"+taskID, map[string]any{
		"windowStartDate": "",
		"windowEndDate":   "2024-01-15",
		"windowEndTime":   "18:00",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, gotInput.WindowStartDate, "WindowStartDate должен быть ptr(\"\") — признак сброса")
	assert.Equal(t, "", *gotInput.WindowStartDate)
}

// 5.3.7b PATCH /api/tasks/:id — сброс конца окна (windowEndDate="") → 200
func TestTaskHandler_Update_ClearWindowEnd(t *testing.T) {
	taskID := uuid.NewString()
	var gotInput task.UpdateInput
	repo := &handlerTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			gotInput = in
			return makeTestTask(taskID, taskTestUserID, "task"), true, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/"+taskID, map[string]any{
		"windowStartDate": "2024-01-15",
		"windowStartTime": "09:00",
		"windowEndDate":   "",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, gotInput.WindowEndDate, "WindowEndDate должен быть ptr(\"\") — признак сброса")
	assert.Equal(t, "", *gotInput.WindowEndDate)
}

// 5.3.8 PATCH /api/tasks/:id — не найдено → 404
func TestTaskHandler_Update_NotFound(t *testing.T) {
	repo := &handlerTaskRepo{
		updateFn: func(_ context.Context, _, _ string, _ task.UpdateInput) (task.Task, bool, error) {
			return task.Task{}, false, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/"+uuid.NewString(), map[string]any{"title": "t"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.3.9 PATCH /api/tasks/:id — чужая задача → 404
func TestTaskHandler_Update_OtherUsersTask(t *testing.T) {
	repo := &handlerTaskRepo{
		updateFn: func(_ context.Context, _, _ string, _ task.UpdateInput) (task.Task, bool, error) {
			return task.Task{}, false, nil // isolation returns not found
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/other-users-task", map[string]any{"title": "t"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.3.10 DELETE /api/tasks/:id — успех
func TestTaskHandler_Delete_Success(t *testing.T) {
	taskID := uuid.NewString()
	repo := &handlerTaskRepo{
		softDeleteFn: func(_ context.Context, _, id string) (bool, error) {
			assert.Equal(t, taskID, id)
			return true, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "DELETE", "/api/tasks/"+taskID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

// 5.3.11 DELETE /api/tasks/:id — не найдено → 404
func TestTaskHandler_Delete_NotFound(t *testing.T) {
	repo := &handlerTaskRepo{
		softDeleteFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "DELETE", "/api/tasks/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 5.3.12 PATCH /api/tasks/order — успех
func TestTaskHandler_Reorder_Success(t *testing.T) {
	id1 := uuid.NewString()
	id2 := uuid.NewString()
	repo := &handlerTaskRepo{
		bulkReorderFn: func(_ context.Context, _ string, in task.ReorderInput) error {
			assert.Len(t, in.Items, 2)
			return nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/order", map[string]any{
		"order": []map[string]any{
			{"id": id1, "sortIndex": 0},
			{"id": id2, "sortIndex": 1},
		},
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

// 5.3.13 PATCH /api/tasks/order — пустой список → 200 (idempotent)
func TestTaskHandler_Reorder_EmptyList(t *testing.T) {
	repo := &handlerTaskRepo{
		bulkReorderFn: func(_ context.Context, _ string, _ task.ReorderInput) error {
			return nil
		},
	}
	r := newTaskTestRouter(repo)
	w := doJSON(t, r, "PATCH", "/api/tasks/order", map[string]any{
		"order": []any{},
	})
	assert.Equal(t, http.StatusOK, w.Code)
}
