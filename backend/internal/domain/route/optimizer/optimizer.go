package optimizer

import "context"

type StopTiming struct {
	NodeIdx           int
	ArrivalSec        int64
	WaitSec           int
	ServiceStartSec   int64
	ServiceEndSec     int64
	TravelFromPrevSec int
}

type Result struct {
	Order   []int
	Timings []StopTiming

	TotalDistanceM  int
	TotalTravelSec  int
	TotalServiceSec int
	TotalWaitSec    int
}

type PrecedencePair struct {
	Before int
	After  int
}

type Constraints struct {
	StartNodeIdx    *int
	EndNodeIdx      *int
	PrecedencePairs []PrecedencePair
}

// Реализации должны быть безопасны для конкурентного использования.
type Optimizer interface {
	Name() string
	Optimize(ctx context.Context, g *Graph, startTimeUnix int64, c Constraints) (Result, error)
}
