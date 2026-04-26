package route

import "time"

// Route — факт построения маршрута.
// Status (draft|optimized|failed) и Source (manual|optimized) проставляются в SQL
// при создании/сохранении и видны во всех ответах API; именно по ним фронт отличает
// черновик от оптимизированного маршрута.
type Route struct {
	ID         string
	UserID     string
	Status     string
	Source     string
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
