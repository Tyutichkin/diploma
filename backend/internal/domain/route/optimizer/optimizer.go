package optimizer

import "context"

// StopTiming holds computed timing for one stop in visit order.
type StopTiming struct {
	NodeIdx           int   // index into Graph.Nodes
	ArrivalSec        int64 // Unix seconds: when the vehicle arrives
	WaitSec           int   // seconds waiting for window to open (0 if no wait)
	ServiceStartSec   int64 // Unix seconds: arrival + wait
	ServiceEndSec     int64 // Unix seconds: service start + DurationMin*60
	TravelFromPrevSec int   // seconds of travel from previous stop (0 for first)
}

// Result holds the output of a route optimization run.
type Result struct {
	// Order contains indices into Graph.Nodes in the recommended visit sequence.
	Order []int

	// Timings holds per-stop timing data, one entry per element of Order (same length).
	Timings []StopTiming

	TotalDistanceM  int // sum of edge distances along the route, metres
	TotalTravelSec  int // sum of edge durations along the route, seconds
	TotalServiceSec int // sum of DurationMin*60 for all visited nodes
	TotalWaitSec    int // time spent waiting for time windows to open, seconds
}

// PrecedencePair задаёт ограничение порядка: узел Before должен быть посещён
// строго до узла After. Оба значения — индексы в Graph.Nodes.
type PrecedencePair struct {
	Before int
	After  int
}

// Constraints содержит дополнительные ограничения на порядок обхода узлов.
// Нулевое значение означает отсутствие ограничений.
type Constraints struct {
	// StartNodeIdx фиксирует первый узел маршрута.
	// Если nil — стартовый узел выбирается алгоритмом.
	StartNodeIdx *int

	// EndNodeIdx фиксирует последний узел маршрута.
	// Если nil — конечный узел не фиксируется.
	EndNodeIdx *int

	// PrecedencePairs задаёт попарные ограничения порядка посещения.
	PrecedencePairs []PrecedencePair
}

// Optimizer finds an optimised visit order for nodes in a graph.
// Implementations must be safe for concurrent use.
type Optimizer interface {
	// Name — короткий идентификатор алгоритма, сохраняется в БД рядом с маршрутом.
	Name() string

	// Optimize возвращает индексы узлов в оптимальном порядке обхода.
	// startTimeUnix — время выезда в секундах Unix (например, 1704085200 = 2024-01-01T09:00:00Z).
	// c — дополнительные ограничения; нулевое значение означает их отсутствие.
	Optimize(ctx context.Context, g *Graph, startTimeUnix int64, c Constraints) (Result, error)
}
