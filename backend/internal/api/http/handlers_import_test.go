package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planner-backend/internal/domain/task"
	taskstory "planner-backend/internal/domain/task/story"
	importstory "planner-backend/internal/domain/taskimport/story"
)

// importBatchRepo — task.Repository с настраиваемым BatchCreate.
type importBatchRepo struct {
	batchFn func(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error)
}

func (r *importBatchRepo) ListByUser(context.Context, string) ([]task.Task, error) {
	return nil, nil
}
func (r *importBatchRepo) GetByIDs(context.Context, string, []string) ([]task.Task, error) {
	return nil, nil
}
func (r *importBatchRepo) Create(context.Context, string, task.CreateInput) (task.Task, error) {
	return task.Task{}, nil
}
func (r *importBatchRepo) BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	if r.batchFn != nil {
		return r.batchFn(ctx, userID, inputs)
	}
	out := make([]task.Task, len(inputs))
	for i, in := range inputs {
		out[i] = task.Task{ID: uuid.NewString(), Title: in.Title, UserID: userID, SortIndex: in.SortIndex}
	}
	return out, nil
}
func (r *importBatchRepo) Update(context.Context, string, string, task.UpdateInput) (task.Task, bool, error) {
	return task.Task{}, false, nil
}
func (r *importBatchRepo) SoftDelete(context.Context, string, string) (bool, error) {
	return false, nil
}
func (r *importBatchRepo) SoftDeleteAll(context.Context, string) (int64, error) {
	return 0, nil
}
func (r *importBatchRepo) BulkReorder(context.Context, string, task.ReorderInput) error {
	return nil
}

// newImportTestRouter строит gin-роутер с /api/tasks/import и bypass-ом авторизации.
func newImportTestRouter(repo *importBatchRepo) *gin.Engine {
	tStory := taskstory.New(repo)
	impStory := importstory.New(tStory)
	importH := NewImportHandlers(impStory)

	r := gin.New()
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set(ctxUserIDKey, taskTestUserID)
		c.Next()
	})
	api.POST("/tasks/import", importH.Import)
	return r
}

// doMultipart посылает multipart/form-data с одним файлом.
func doMultipart(t *testing.T, r *gin.Engine, path, fieldName, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest("POST", path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestImportHandler_CSV_Success(t *testing.T) {
	var captured []task.CreateInput
	repo := &importBatchRepo{
		batchFn: func(_ context.Context, _ string, inputs []task.CreateInput) ([]task.Task, error) {
			captured = inputs
			out := make([]task.Task, len(inputs))
			for i, in := range inputs {
				out[i] = task.Task{ID: "id-" + in.Title, Title: in.Title, SortIndex: in.SortIndex}
			}
			return out, nil
		},
	}
	r := newImportTestRouter(repo)

	csv := []byte("title,duration\nA,30\nB,15\n")
	w := doMultipart(t, r, "/api/tasks/import?startSortIndex=7", "file", "tasks.csv", csv)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Imported []task.Task         `json:"imported"`
		Errors   []ImportRowErrorDTO `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Imported, 2)
	assert.Empty(t, resp.Errors)

	// sort_index учитывает startSortIndex
	require.Len(t, captured, 2)
	assert.Equal(t, 7, captured[0].SortIndex)
	assert.Equal(t, 8, captured[1].SortIndex)
}

func TestImportHandler_MissingFileField(t *testing.T) {
	r := newImportTestRouter(&importBatchRepo{})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("other", "val"))
	require.NoError(t, mw.Close())

	req := httptest.NewRequest("POST", "/api/tasks/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "file")
}

func TestImportHandler_UnsupportedExtension(t *testing.T) {
	r := newImportTestRouter(&importBatchRepo{})
	w := doMultipart(t, r, "/api/tasks/import", "file", "data.txt", []byte("title\nA\n"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "unsupported")
}

func TestImportHandler_CSV_ReturnsRowErrors(t *testing.T) {
	r := newImportTestRouter(&importBatchRepo{})
	csv := []byte("title,duration\n,10\nA,abc\n")
	w := doMultipart(t, r, "/api/tasks/import", "file", "t.csv", csv)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Imported []task.Task         `json:"imported"`
		Errors   []ImportRowErrorDTO `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Imported)
	require.Len(t, resp.Errors, 2)
	assert.Equal(t, 1, resp.Errors[0].Row)
	assert.Equal(t, 2, resp.Errors[1].Row)
}

func TestImportHandler_Unauthorized(t *testing.T) {
	tStory := taskstory.New(&importBatchRepo{})
	impStory := importstory.New(tStory)
	importH := NewImportHandlers(impStory)
	r := gin.New()
	r.POST("/api/tasks/import", importH.Import)

	w := doMultipart(t, r, "/api/tasks/import", "file", "t.csv", []byte("title\nA\n"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
