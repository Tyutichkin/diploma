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

type DistanceCellDTO struct {
	DistanceM   int `json:"distanceM"`
	DurationSec int `json:"durationSec"`
}

type PrecedencePairDTO struct {
	BeforeTaskID string `json:"beforeTaskId"`
	AfterTaskID  string `json:"afterTaskId"`
}

type ImportRowErrorDTO struct {
	Row    int      `json:"row"`
	Title  string   `json:"title"`
	Errors []string `json:"errors"`
}

type ImportTasksResp struct {
	Imported any                 `json:"imported"`
	Errors   []ImportRowErrorDTO `json:"errors"`
}

type OptimizeRouteReq struct {
	TaskIDs               []string            `json:"taskIds"`
	StartTimeUnix         int64               `json:"startTimeUnix"`
	DistanceMatrix        [][]DistanceCellDTO `json:"distanceMatrix,omitempty"` // Если DistanceMatrix передана, бэкенд не ходит в OSRM, а берет данные из Yandex.
	StartTaskID           *string             `json:"startTaskId,omitempty"`
	EndTaskID             *string             `json:"endTaskId,omitempty"`
	PrecedenceConstraints []PrecedencePairDTO `json:"precedenceConstraints,omitempty"`
}
