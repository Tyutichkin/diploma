package gate

import (
	"context"

	"planner-backend/internal/domain/route"
)

// StopInput carries the data needed to persist one route stop.
type StopInput struct {
	TaskID            string
	Position          int
	TravelFromPrevSec int // 0 for the first stop
}

type Repository interface {
	CreateDraft(ctx context.Context, userID, source string) (route.Route, error)
	ListByUser(ctx context.Context, userID string) ([]route.Route, error)
	ReplaceStops(ctx context.Context, routeID string, taskIDs []string) error
	GetFull(ctx context.Context, userID, routeID string) (route.Full, bool, error)
	MarkFailed(ctx context.Context, routeID string) error

	// SaveOptimizedRoute creates a fully optimised route (record + stops + stats)
	// in a single transaction and returns the created route.
	SaveOptimizedRoute(
		ctx context.Context,
		userID, algorithm string,
		stops []StopInput,
		distanceM, travelSec, serviceSec, waitSec int,
	) (route.Route, error)

	// DeleteRoute removes a single route belonging to userID.
	// Returns false when the route does not exist or belongs to another user.
	DeleteRoute(ctx context.Context, userID, routeID string) (bool, error)

	// DeleteAllRoutes removes every route belonging to userID.
	DeleteAllRoutes(ctx context.Context, userID string) error

	// RenameRoute updates the human-readable name of a route.
	// Returns false when the route does not exist or belongs to another user.
	RenameRoute(ctx context.Context, userID, routeID, name string) (bool, error)
}
