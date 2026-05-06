package gate

import (
	"context"
	"time"

	"planner-backend/internal/domain/route"
)

type StopInput struct {
	TaskID            string
	Position          int
	TravelFromPrevSec int
	ArriveTime        *time.Time
	ServiceStartTime  *time.Time
	ServiceEndTime    *time.Time
	WaitSec           *int
}

// Один маршрут на пользователя: SaveOptimizedRoute перезаписывает предыдущий.
type Repository interface {
	SaveOptimizedRoute(
		ctx context.Context,
		userID, algorithm string,
		stops []StopInput,
		distanceM, travelSec, serviceSec, waitSec int,
	) (route.Route, error)
}
