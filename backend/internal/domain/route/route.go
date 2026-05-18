package route

import "time"

type Route struct {
	ID              string
	UserID          string
	Algorithm       string
	TotalDistanceM  int
	TotalTravelSec  int
	TotalServiceSec int
	TotalWaitSec    int
	ComputedAt      time.Time
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
