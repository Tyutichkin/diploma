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

type Repository interface {
	CreateDraft(ctx context.Context, userID, source string) (route.Route, error)
	ListByUser(ctx context.Context, userID string) ([]route.Route, error)
	ReplaceStops(ctx context.Context, routeID string, taskIDs []string) error
	GetFull(ctx context.Context, userID, routeID string) (route.Full, bool, error)
	MarkFailed(ctx context.Context, routeID string) error

	// SaveOptimizedRoute сохраняет маршрут, остановки и статистику в одной транзакции.
	SaveOptimizedRoute(
		ctx context.Context,
		userID, algorithm string,
		stops []StopInput,
		distanceM, travelSec, serviceSec, waitSec int,
	) (route.Route, error)

	// DeleteRoute — false если маршрут не найден или принадлежит другому пользователю.
	DeleteRoute(ctx context.Context, userID, routeID string) (bool, error)

	DeleteAllRoutes(ctx context.Context, userID string) error

	// RenameRoute — false если маршрут не найден или принадлежит другому пользователю.
	RenameRoute(ctx context.Context, userID, routeID, name string) (bool, error)
}
