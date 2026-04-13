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
	Title       string  `json:"title"`
	AddressText string  `json:"addressText"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DurationMin int     `json:"durationMin"`
	WindowStart *string `json:"windowStart"`
	WindowEnd   *string `json:"windowEnd"`
	SortIndex   int     `json:"sortIndex"`
}

type UpdateTaskReq struct {
	Title       *string  `json:"title"`
	AddressText *string  `json:"addressText"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	DurationMin *int     `json:"durationMin"`
	WindowStart *string  `json:"windowStart"`
	WindowEnd   *string  `json:"windowEnd"`
	SortIndex   *int     `json:"sortIndex"`
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

// DistanceCellDTO is one cell of the client-supplied distance matrix.
type DistanceCellDTO struct {
	DistanceM   int `json:"distanceM"`   // travel distance in metres
	DurationSec int `json:"durationSec"` // travel duration in seconds
}

// PrecedencePairDTO задаёт ограничение порядка: задача BeforeTaskId
// должна быть выполнена строго до задачи AfterTaskId.
type PrecedencePairDTO struct {
	BeforeTaskID string `json:"beforeTaskId"`
	AfterTaskID  string `json:"afterTaskId"`
}

// OptimizeRouteReq — тело POST /api/routes/optimize.
// DistanceMatrix желательно передавать: тогда бэкенд не ходит в OSRM
// и порядок задач будет согласован с маршрутом на карте (Яндекс).
type OptimizeRouteReq struct {
	TaskIDs               []string            `json:"taskIds"`
	StartTimeMins         int                 `json:"startTimeMins"`             // минуты от полуночи, по умолчанию 540 (09:00)
	DistanceMatrix        [][]DistanceCellDTO `json:"distanceMatrix,omitempty"`  // matrix[i][j] — стоимость TaskIDs[i]→TaskIDs[j]
	StartTaskID           *string             `json:"startTaskId,omitempty"`     // фиксированная первая точка маршрута
	EndTaskID             *string             `json:"endTaskId,omitempty"`       // фиксированная последняя точка маршрута
	PrecedenceConstraints []PrecedencePairDTO `json:"precedenceConstraints,omitempty"` // ограничения порядка выполнения
}
