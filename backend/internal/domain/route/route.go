package route

import "time"

// Route — факт построения маршрута.
type Route struct {
	ID         string
	UserID     string
	Status     string // draft|optimized|failed TODO: какой практический смысл несет это поле? Если его нет, то удалить
	Source     string // manual|optimized TODO: какой практический смысл несет это поле? Если его нет, то удалить
	Name       *string
	Algorithm  *string
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
}

type Stop struct {
	ID                string
	RouteID           string
	TaskID            string
	Position          int
	TravelFromPrevSec *int
	ArriveTime        *time.Time
	ServiceStartTime  *time.Time
	ServiceEndTime    *time.Time
	WaitSec           *int
}

type Stats struct {
	RouteID         string
	TotalDistanceM  *int
	TotalTravelSec  *int
	TotalServiceSec *int
	TotalWaitSec    *int
	ComputedAt      time.Time
}

type Geometry struct {
	RouteID   string
	Polyline  string
	BBoxJSON  *string
	GeoJSON   *string
	UpdatedAt time.Time
}
