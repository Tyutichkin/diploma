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
type Optimizer interface {
	// Name — короткий идентификатор алгоритма, сохраняется в БД рядом с маршрутом.
	Name() string

	// Optimize возвращает индексы узлов в оптимальном порядке обхода.
	// startTimeMins — время выезда в минутах от полуночи (например, 540 = 09:00).
	Optimize(ctx context.Context, g *Graph, startTimeMins int) (Result, error)
}
