package gate

import (
	"context"
	"time"

	"planner-backend/internal/domain/route"
)

// StopInput — данные для сохранения одной остановки маршрута.
type StopInput struct {
	TaskID            string
	Position          int
	TravelFromPrevSec int
	ArriveTime        *time.Time
	ServiceStartTime  *time.Time
	ServiceEndTime    *time.Time
	WaitSec           *int // секунды ожидания открытия окна
}

// Repository хранит ровно один маршрут на пользователя: каждое сохранение
// перезаписывает предыдущий (DELETE + INSERT в одной транзакции).
type Repository interface {
	// SaveOptimizedRoute сохраняет маршрут, остановки и статистику в одной транзакции.
	// Если у пользователя уже был маршрут — он удаляется (CASCADE сносит stops/stats/geometry).
	SaveOptimizedRoute(
		ctx context.Context,
		userID, algorithm string,
		stops []StopInput,
		distanceM, travelSec, serviceSec, waitSec int,
	) (route.Route, error)
}
