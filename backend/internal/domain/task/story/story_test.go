package story

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planner-backend/internal/domain/task"
)

// ── mock repository ───────────────────────────────────────────────────────────

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
func (m *mockTaskRepo) BulkReorder(ctx context.Context, userID string, in task.ReorderInput) error {
	return m.bulkReorderFn(ctx, userID, in)
}

func intPtr(v int) *int { return &v }

// ── helpers ───────────────────────────────────────────────────────────────────

func makeTask(id, userID, title string) task.Task {
	lat := 55.75
	lon := 37.61
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		AddressText: "Some Address",
		Latitude:    &lat,
		Longitude:   &lon,
		DurationMin: intPtr(30),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// makeTaskNoAddr создаёт задачу без геопривязки.
func makeTaskNoAddr(id, userID, title string) task.Task {
	return task.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		AddressText: "",
		Latitude:    nil,
		Longitude:   nil,
		DurationMin: intPtr(15),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func validCreateInput() task.CreateInput {
	lat := 55.75
	lon := 37.61
	return task.CreateInput{
		Title:       "Buy groceries",
		AddressText: "123 Main St",
		Latitude:    &lat,
		Longitude:   &lon,
		DurationMin: intPtr(30),
	}
}

// ── Create tests ──────────────────────────────────────────────────────────────

// 2.1.1 Успешное создание
func TestCreate_Success(t *testing.T) {
	uid := uuid.NewString()
	expected := makeTask(uuid.NewString(), uid, "Buy groceries")

	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, _ task.CreateInput) (task.Task, error) {
			return expected, nil
		},
	}
	s := New(repo)
	got, err := s.Create(context.Background(), uid, validCreateInput())
	require.NoError(t, err)
	assert.Equal(t, expected.ID, got.ID)
}

// 2.1.2 Пустой title
func TestCreate_EmptyTitle(t *testing.T) {
	s := New(&mockTaskRepo{})
	in := validCreateInput()
	in.Title = ""
	_, err := s.Create(context.Background(), "uid", in)
	assert.Error(t, err)
}

// 2.1.3 Задача без адреса — создаётся успешно (адрес необязателен)
func TestCreate_NoAddress(t *testing.T) {
	uid := uuid.NewString()
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			assert.Equal(t, "", in.AddressText)
			assert.Nil(t, in.Latitude)
			assert.Nil(t, in.Longitude)
			return makeTaskNoAddr(uuid.NewString(), uid, in.Title), nil
		},
	}
	s := New(repo)
	in := task.CreateInput{
		Title:       "Позвонить клиенту",
		AddressText: "",
		Latitude:    nil,
		Longitude:   nil,
		DurationMin: intPtr(15),
	}
	_, err := s.Create(context.Background(), uid, in)
	require.NoError(t, err)
}

// 2.1.4 Duration = nil — мгновенная задача, создаётся успешно
func TestCreate_NilDuration(t *testing.T) {
	uid := uuid.NewString()
	expected := makeTask(uuid.NewString(), uid, "Buy groceries")
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, _ task.CreateInput) (task.Task, error) {
			return expected, nil
		},
	}
	s := New(repo)
	in := validCreateInput()
	in.DurationMin = nil
	_, err := s.Create(context.Background(), uid, in)
	assert.NoError(t, err)
}

// 2.1.5 Duration отрицательная
func TestCreate_NegativeDuration(t *testing.T) {
	s := New(&mockTaskRepo{})
	in := validCreateInput()
	in.DurationMin = intPtr(-1)
	_, err := s.Create(context.Background(), "uid", in)
	assert.Error(t, err)
}

// 2.1.6 Без time window — задача создаётся без ограничений
func TestCreate_NoTimeWindow(t *testing.T) {
	uid := uuid.NewString()
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			assert.Nil(t, in.WindowStartDate)
			assert.Nil(t, in.WindowEndDate)
			return makeTask(uuid.NewString(), uid, in.Title), nil
		},
	}
	s := New(repo)
	in := validCreateInput()
	in.WindowStartDate = nil
	in.WindowStartTime = nil
	in.WindowEndDate = nil
	in.WindowEndTime = nil
	_, err := s.Create(context.Background(), uid, in)
	require.NoError(t, err)
}

// 2.1.7 Корректное временное окно — задача создаётся успешно
func TestCreate_ValidTimeWindow(t *testing.T) {
	uid := uuid.NewString()
	wsDate := "2024-01-15"
	wsTime := "09:00"
	weDate := "2024-01-15"
	weTime := "18:00"
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			return makeTask(uuid.NewString(), uid, in.Title), nil
		},
	}
	s := New(repo)
	in := validCreateInput()
	in.WindowStartDate = &wsDate
	in.WindowStartTime = &wsTime
	in.WindowEndDate = &weDate
	in.WindowEndTime = &weTime
	_, err := s.Create(context.Background(), uid, in)
	require.NoError(t, err)
}

// 2.1.8 WindowEnd < WindowStart — story должна отклонять
func TestCreate_TimeWindowEndBeforeStart(t *testing.T) {
	s := New(&mockTaskRepo{})
	wsDate := "2024-01-15"
	wsTime := "18:00"
	weDate := "2024-01-15"
	weTime := "09:00"
	in := validCreateInput()
	in.WindowStartDate = &wsDate
	in.WindowStartTime = &wsTime
	in.WindowEndDate = &weDate
	in.WindowEndTime = &weTime
	_, err := s.Create(context.Background(), "uid", in)
	assert.Error(t, err)
}

// 2.1.9 Только начало окна без конца — допустимо
func TestCreate_OnlyWindowStart(t *testing.T) {
	uid := uuid.NewString()
	wsDate := "2024-01-15"
	wsTime := "09:00"
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, in task.CreateInput) (task.Task, error) {
			return makeTask(uuid.NewString(), uid, in.Title), nil
		},
	}
	s := New(repo)
	in := validCreateInput()
	in.WindowStartDate = &wsDate
	in.WindowStartTime = &wsTime
	in.WindowEndDate = nil
	in.WindowEndTime = nil
	_, err := s.Create(context.Background(), uid, in)
	require.NoError(t, err)
}

// 2.2.4 Обновление с инвертированным окном — story должна отклонять
func TestUpdate_TimeWindowEndBeforeStart(t *testing.T) {
	s := New(&mockTaskRepo{})
	wsDate := "2024-01-15"
	wsTime := "20:00"
	weDate := "2024-01-15"
	weTime := "08:00"
	_, _, err := s.Update(context.Background(), "uid", uuid.NewString(), task.UpdateInput{
		WindowStartDate: &wsDate,
		WindowStartTime: &wsTime,
		WindowEndDate:   &weDate,
		WindowEndTime:   &weTime,
	})
	assert.Error(t, err)
}

// 2.2.5 Обновление только конца окна без начала — допустимо
func TestUpdate_OnlyWindowEnd(t *testing.T) {
	uid := uuid.NewString()
	taskID := uuid.NewString()
	weDate := "2024-01-15"
	weTime := "18:00"
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			return makeTask(taskID, uid, "task"), true, nil
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), uid, taskID, task.UpdateInput{WindowEndDate: &weDate, WindowEndTime: &weTime})
	require.NoError(t, err)
	assert.True(t, found)
}

// 2.2.6 Сброс даты начала окна (пустая строка) — допустимо, repo вызывается с ptr("")
func TestUpdate_ClearWindowStartDate(t *testing.T) {
	uid := uuid.NewString()
	taskID := uuid.NewString()
	empty := ""
	weDate := "2024-01-15"
	weTime := "18:00"
	var gotInput task.UpdateInput
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			gotInput = in
			return makeTask(taskID, uid, "task"), true, nil
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), uid, taskID, task.UpdateInput{
		WindowStartDate: &empty,
		WindowEndDate:   &weDate,
		WindowEndTime:   &weTime,
	})
	require.NoError(t, err)
	assert.True(t, found)
	require.NotNil(t, gotInput.WindowStartDate)
	assert.Equal(t, "", *gotInput.WindowStartDate)
}

// 2.2.7 Сброс даты конца окна (пустая строка) — допустимо, repo вызывается с ptr("")
func TestUpdate_ClearWindowEndDate(t *testing.T) {
	uid := uuid.NewString()
	taskID := uuid.NewString()
	wsDate := "2024-01-15"
	wsTime := "09:00"
	empty := ""
	var gotInput task.UpdateInput
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			gotInput = in
			return makeTask(taskID, uid, "task"), true, nil
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), uid, taskID, task.UpdateInput{
		WindowStartDate: &wsDate,
		WindowStartTime: &wsTime,
		WindowEndDate:   &empty,
	})
	require.NoError(t, err)
	assert.True(t, found)
	require.NotNil(t, gotInput.WindowEndDate)
	assert.Equal(t, "", *gotInput.WindowEndDate)
}

// 2.2.8 Сброс обоих полей окна — допустимо
func TestUpdate_ClearBothWindows(t *testing.T) {
	uid := uuid.NewString()
	taskID := uuid.NewString()
	empty := ""
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			return makeTask(taskID, uid, "task"), true, nil
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), uid, taskID, task.UpdateInput{
		WindowStartDate: &empty,
		WindowStartTime: &empty,
		WindowEndDate:   &empty,
		WindowEndTime:   &empty,
	})
	require.NoError(t, err)
	assert.True(t, found)
}

// ── Update tests ──────────────────────────────────────────────────────────────

// 2.2.1 Частичное обновление
func TestUpdate_PartialUpdate(t *testing.T) {
	uid := uuid.NewString()
	taskID := uuid.NewString()
	newTitle := "Updated Title"

	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, in task.UpdateInput) (task.Task, bool, error) {
			t := makeTask(taskID, uid, *in.Title)
			return t, true, nil
		},
	}
	s := New(repo)
	in := task.UpdateInput{Title: &newTitle}
	got, found, err := s.Update(context.Background(), uid, taskID, in)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, newTitle, got.Title)
}

// 2.2.2 Обновление несуществующей задачи
func TestUpdate_TaskNotFound(t *testing.T) {
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, _ task.UpdateInput) (task.Task, bool, error) {
			return task.Task{}, false, nil
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), "uid", uuid.NewString(), task.UpdateInput{})
	require.NoError(t, err)
	assert.False(t, found)
}

// 2.2.3 Обновление чужой задачи
func TestUpdate_AnotherUsersTask(t *testing.T) {
	repo := &mockTaskRepo{
		updateFn: func(_ context.Context, _, _ string, _ task.UpdateInput) (task.Task, bool, error) {
			return task.Task{}, false, nil // isolation by userID
		},
	}
	s := New(repo)
	_, found, err := s.Update(context.Background(), "user-a", "task-of-user-b", task.UpdateInput{})
	require.NoError(t, err)
	assert.False(t, found)
}

// ── Delete (SoftDelete) tests ─────────────────────────────────────────────────

// 2.3.1 Успешное удаление
func TestDelete_Success(t *testing.T) {
	repo := &mockTaskRepo{
		softDeleteFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	s := New(repo)
	found, err := s.Delete(context.Background(), "uid", uuid.NewString())
	require.NoError(t, err)
	assert.True(t, found)
}

// 2.3.2 Удаление несуществующей задачи
func TestDelete_TaskNotFound(t *testing.T) {
	repo := &mockTaskRepo{
		softDeleteFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
	}
	s := New(repo)
	found, err := s.Delete(context.Background(), "uid", uuid.NewString())
	require.NoError(t, err)
	assert.False(t, found)
}

// 2.3.3 Удаление чужой задачи
func TestDelete_AnotherUsersTask(t *testing.T) {
	repo := &mockTaskRepo{
		softDeleteFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil // isolation
		},
	}
	s := New(repo)
	found, err := s.Delete(context.Background(), "user-a", "task-of-user-b")
	require.NoError(t, err)
	assert.False(t, found)
}

// 2.3.4 Повторное удаление
func TestDelete_Twice(t *testing.T) {
	callCount := 0
	repo := &mockTaskRepo{
		softDeleteFn: func(_ context.Context, _, _ string) (bool, error) {
			callCount++
			if callCount == 1 {
				return true, nil
			}
			return false, nil // second call: already deleted
		},
	}
	s := New(repo)
	taskID := uuid.NewString()
	found1, _ := s.Delete(context.Background(), "uid", taskID)
	found2, _ := s.Delete(context.Background(), "uid", taskID)
	assert.True(t, found1)
	assert.False(t, found2)
}

// ── Reorder tests ─────────────────────────────────────────────────────────────

// 2.4.1 Успешная перестановка
func TestReorder_Success(t *testing.T) {
	repo := &mockTaskRepo{
		bulkReorderFn: func(_ context.Context, _ string, _ task.ReorderInput) error {
			return nil
		},
	}
	s := New(repo)
	in := task.ReorderInput{
		Items: []task.ReorderItem{
			{TaskID: uuid.NewString(), SortIndex: 0},
			{TaskID: uuid.NewString(), SortIndex: 1},
		},
	}
	err := s.Reorder(context.Background(), "uid", in)
	assert.NoError(t, err)
}

// 2.4.2 Пустой список
func TestReorder_EmptyList(t *testing.T) {
	called := false
	repo := &mockTaskRepo{
		bulkReorderFn: func(_ context.Context, _ string, _ task.ReorderInput) error {
			called = true
			return nil
		},
	}
	s := New(repo)
	err := s.Reorder(context.Background(), "uid", task.ReorderInput{Items: []task.ReorderItem{}})
	assert.NoError(t, err)
	assert.True(t, called) // story delegates to repo even for empty list
}

// 2.4.3 Ошибка репозитория
func TestReorder_RepoError(t *testing.T) {
	repo := &mockTaskRepo{
		bulkReorderFn: func(_ context.Context, _ string, _ task.ReorderInput) error {
			return errors.New("db error")
		},
	}
	s := New(repo)
	err := s.Reorder(context.Background(), "uid", task.ReorderInput{Items: []task.ReorderItem{{TaskID: "id", SortIndex: 0}}})
	assert.Error(t, err)
}

// ── List tests ────────────────────────────────────────────────────────────────

// 2.5.1 Возвращает только задачи текущего пользователя
func TestList_ReturnsOnlyUserTasks(t *testing.T) {
	uid := uuid.NewString()
	tasks := []task.Task{
		makeTask(uuid.NewString(), uid, "Task A"),
		makeTask(uuid.NewString(), uid, "Task B"),
	}
	repo := &mockTaskRepo{
		listByUserFn: func(_ context.Context, id string) ([]task.Task, error) {
			assert.Equal(t, uid, id)
			return tasks, nil
		},
	}
	s := New(repo)
	got, err := s.List(context.Background(), uid)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

// 2.5.3 Пустой список
func TestList_Empty(t *testing.T) {
	repo := &mockTaskRepo{
		listByUserFn: func(_ context.Context, _ string) ([]task.Task, error) {
			return []task.Task{}, nil
		},
	}
	s := New(repo)
	got, err := s.List(context.Background(), "uid")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// 2.5.4 Сортировка по sort_index (repo contract)
func TestList_SortedBySortIndex(t *testing.T) {
	uid := uuid.NewString()
	t1 := makeTask(uuid.NewString(), uid, "First")
	t1.SortIndex = 0
	t2 := makeTask(uuid.NewString(), uid, "Second")
	t2.SortIndex = 1
	t3 := makeTask(uuid.NewString(), uid, "Third")
	t3.SortIndex = 2

	repo := &mockTaskRepo{
		listByUserFn: func(_ context.Context, _ string) ([]task.Task, error) {
			return []task.Task{t1, t2, t3}, nil
		},
	}
	s := New(repo)
	got, err := s.List(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, 0, got[0].SortIndex)
	assert.Equal(t, 1, got[1].SortIndex)
	assert.Equal(t, 2, got[2].SortIndex)
}

// ── Error propagation ─────────────────────────────────────────────────────────

func TestList_RepoError(t *testing.T) {
	repo := &mockTaskRepo{
		listByUserFn: func(_ context.Context, _ string) ([]task.Task, error) {
			return nil, errors.New("db error")
		},
	}
	s := New(repo)
	_, err := s.List(context.Background(), "uid")
	assert.Error(t, err)
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, _ string, _ task.CreateInput) (task.Task, error) {
			return task.Task{}, errors.New("db error")
		},
	}
	s := New(repo)
	_, err := s.Create(context.Background(), "uid", validCreateInput())
	assert.Error(t, err)
}
