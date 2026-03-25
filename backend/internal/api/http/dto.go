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

// OptimizeRouteReq is the request body for POST /api/routes/optimize.
//
// The client should populate DistanceMatrix using the same routing engine
// (Yandex Maps) that is used for map visualisation so that the optimised order
// is consistent with the route drawn on screen.
//
// If DistanceMatrix is omitted the backend falls back to querying the
// configured external routing API (OSRM by default).
type OptimizeRouteReq struct {
	// TaskIDs are the IDs of the tasks to include in the route (order irrelevant).
	TaskIDs []string `json:"taskIds"`
	// StartTimeMins is the departure time in minutes since midnight (e.g. 540 = 09:00).
	// Defaults to 540 when omitted or zero.
	StartTimeMins int `json:"startTimeMins"`
	// DistanceMatrix[i][j] is the travel cost from TaskIDs[i] to TaskIDs[j].
	// When provided the backend skips the external API call and uses this data directly.
	DistanceMatrix [][]DistanceCellDTO `json:"distanceMatrix,omitempty"`
}
