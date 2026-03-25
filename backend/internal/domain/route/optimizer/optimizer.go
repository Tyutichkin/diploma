package optimizer

import "context"

// Result holds the output of a route optimization run.
type Result struct {
	// Order contains indices into Graph.Nodes in the recommended visit sequence.
	Order []int

	TotalDistanceM  int // sum of edge distances along the route, metres
	TotalTravelSec  int // sum of edge durations along the route, seconds
	TotalServiceSec int // sum of DurationMin*60 for all visited nodes
	TotalWaitSec    int // time spent waiting for time windows to open, seconds
}

// Optimizer finds an optimised visit order for nodes in a graph.
// Implementations must be safe for concurrent use.
//
// To swap the algorithm, provide a different Optimizer implementation
// to routestory.New — no other code needs to change.
type Optimizer interface {
	// Name returns a short, stable identifier for the algorithm.
	// It is persisted in the database alongside the route.
	Name() string

	// Optimize returns node indices in the optimised visit order,
	// together with aggregate travel statistics.
	// startTimeMins is the departure time in minutes since midnight (e.g. 540 = 09:00).
	Optimize(ctx context.Context, g *Graph, startTimeMins int) (Result, error)
}
