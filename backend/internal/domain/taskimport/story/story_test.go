package story

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planner-backend/internal/domain/task"
	taskstory "planner-backend/internal/domain/task/story"
)

type mockTaskRepo struct {
	batchCreateFn func(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error)
}

func (m *mockTaskRepo) ListByUser(context.Context, string) ([]task.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) GetByIDs(context.Context, string, []string) ([]task.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) Create(context.Context, string, task.CreateInput) (task.Task, error) {
	return task.Task{}, nil
}
func (m *mockTaskRepo) BatchCreate(ctx context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, userID, inputs)
	}
	return nil, nil
}
func (m *mockTaskRepo) Update(context.Context, string, string, task.UpdateInput) (task.Task, bool, error) {
	return task.Task{}, false, nil
}
func (m *mockTaskRepo) SoftDelete(context.Context, string, string) (bool, error) {
	return false, nil
}
func (m *mockTaskRepo) BulkReorder(context.Context, string, task.ReorderInput) error {
	return nil
}

func newTestStory(repo *mockTaskRepo) *Story {
	return New(taskstory.New(repo))
}

func TestImport_UnsupportedFormat(t *testing.T) {
	s := newTestStory(&mockTaskRepo{})
	_, err := s.Import(context.Background(), "u1", []byte("x"), "file.txt", 0)
	require.Error(t, err)
	var valErr *task.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Contains(t, valErr.Message, "unsupported")
}

func TestImport_CSV_HappyPath(t *testing.T) {
	csv := []byte("title,duration,window_start,window_end\n" +
		"A,30,09:00,10:00\n" +
		"B,15,,\n")

	var captured []task.CreateInput
	repo := &mockTaskRepo{
		batchCreateFn: func(_ context.Context, userID string, inputs []task.CreateInput) ([]task.Task, error) {
			assert.Equal(t, "u1", userID)
			captured = inputs
			out := make([]task.Task, len(inputs))
			for i, in := range inputs {
				out[i] = task.Task{ID: "id-" + in.Title, Title: in.Title}
			}
			return out, nil
		},
	}
	s := newTestStory(repo)

	res, err := s.Import(context.Background(), "u1", csv, "tasks.csv", 5)
	require.NoError(t, err)
	assert.Len(t, res.Imported, 2)
	assert.Empty(t, res.Errors)

	require.Len(t, captured, 2)
	assert.Equal(t, "A", captured[0].Title)
	assert.Equal(t, 5, captured[0].SortIndex) // startSortIndex
	assert.Equal(t, 6, captured[1].SortIndex)
	require.NotNil(t, captured[0].DurationMin)
	assert.Equal(t, 30, *captured[0].DurationMin)
}

func TestImport_CSV_MixedValidAndInvalid(t *testing.T) {
	csv := []byte("title,duration\nA,30\n,10\nB,abc\nC,5\n")

	repo := &mockTaskRepo{
		batchCreateFn: func(_ context.Context, _ string, inputs []task.CreateInput) ([]task.Task, error) {
			out := make([]task.Task, len(inputs))
			for i, in := range inputs {
				out[i] = task.Task{Title: in.Title}
			}
			return out, nil
		},
	}
	s := newTestStory(repo)

	res, err := s.Import(context.Background(), "u1", csv, "t.csv", 0)
	require.NoError(t, err)
	assert.Len(t, res.Imported, 2) // A, C
	assert.Len(t, res.Errors, 2)   // пустой title, нечисловой duration

	// порядок ошибок — по исходным строкам
	assert.Equal(t, 2, res.Errors[0].Row)
	assert.Equal(t, 3, res.Errors[1].Row)
}

func TestImport_AllRowsInvalid_NoBatchCall(t *testing.T) {
	csv := []byte("title,duration\n,10\n,20\n")
	called := false
	repo := &mockTaskRepo{
		batchCreateFn: func(_ context.Context, _ string, _ []task.CreateInput) ([]task.Task, error) {
			called = true
			return nil, nil
		},
	}
	s := newTestStory(repo)

	res, err := s.Import(context.Background(), "u1", csv, "t.csv", 0)
	require.NoError(t, err)
	assert.False(t, called, "BatchCreate не должен вызываться если нет валидных строк")
	assert.Empty(t, res.Imported)
	assert.Len(t, res.Errors, 2)
}
