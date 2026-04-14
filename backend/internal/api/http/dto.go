package http

type RegisterReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

type TokenPairResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type CreateTaskReq struct {
	Title       string   `json:"title"`
	AddressText *string  `json:"addressText"` // null = нет адреса
	Latitude    *float64 `json:"latitude"`    // null = нет координат
	Longitude   *float64 `json:"longitude"`   // null = нет координат
	DurationMin *int     `json:"durationMin"` // null = мгновенная задача

	WindowStartDate *string `json:"windowStartDate"` // "YYYY-MM-DD"
	WindowStartTime *string `json:"windowStartTime"` // "HH:MM"
	WindowEndDate   *string `json:"windowEndDate"`   // "YYYY-MM-DD"
	WindowEndTime   *string `json:"windowEndTime"`   // "HH:MM"

	SortIndex int `json:"sortIndex"`
}

type BatchCreateTasksReq struct {
	Tasks []CreateTaskReq `json:"tasks"`
}

type UpdateTaskReq struct {
	Title       *string  `json:"title"`
	AddressText *string  `json:"addressText"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	DurationMin *int     `json:"durationMin"`

	WindowStartDate *string `json:"windowStartDate"`
	WindowStartTime *string `json:"windowStartTime"`
	WindowEndDate   *string `json:"windowEndDate"`
	WindowEndTime   *string `json:"windowEndTime"`

	SortIndex   *int  `json:"sortIndex"`
	IsCompleted *bool `json:"isCompleted"`
}

type TaskOrderItem struct {
	ID        string `json:"id"`
	SortIndex int    `json:"sortIndex"`
}

type ReorderTasksReq struct {
	Order []TaskOrderItem `json:"order"`
}

type CreateRouteReq struct {
	Source         string   `json:"source"`
	OrderedTaskIDs []string `json:"orderedTaskIds"`
}

type RenameRouteReq struct {
	Name string `json:"name"`
}

// DistanceCellDTO — ячейка матрицы расстояний от клиента.
type DistanceCellDTO struct {
	DistanceM   int `json:"distanceM"`
	DurationSec int `json:"durationSec"`
}

// PrecedencePairDTO задаёт ограничение порядка: задача BeforeTaskId
// должна быть выполнена строго до задачи AfterTaskId.
type PrecedencePairDTO struct {
	BeforeTaskID string `json:"beforeTaskId"`
	AfterTaskID  string `json:"afterTaskId"`
}

// ImportRowErrorDTO — ошибки отдельной строки файла импорта.
type ImportRowErrorDTO struct {
	Row    int      `json:"row"`
	Title  string   `json:"title"`
	Errors []string `json:"errors"`
}

// ImportTasksResp — ответ POST /api/tasks/import.
type ImportTasksResp struct {
	Imported any                 `json:"imported"` // []task.Task; any чтобы не тащить domain в dto
	Errors   []ImportRowErrorDTO `json:"errors"`
}

// OptimizeRouteReq — тело POST /api/routes/optimize.
// DistanceMatrix желательно передавать: тогда бэкенд не ходит в OSRM
// и порядок задач будет согласован с маршрутом на карте (Яндекс).
type OptimizeRouteReq struct {
	TaskIDs               []string            `json:"taskIds"`
	StartTimeUnix         int64               `json:"startTimeUnix"`                   // Unix-секунды; 0 → сервер вычисляет из задач
	DistanceMatrix        [][]DistanceCellDTO `json:"distanceMatrix,omitempty"`        // matrix[i][j] — стоимость TaskIDs[i]→TaskIDs[j]
	StartTaskID           *string             `json:"startTaskId,omitempty"`           // фиксированная первая точка маршрута
	EndTaskID             *string             `json:"endTaskId,omitempty"`             // фиксированная последняя точка маршрута
	PrecedenceConstraints []PrecedencePairDTO `json:"precedenceConstraints,omitempty"` // ограничения порядка выполнения
}
