package distance

import "context"

type Point struct {
	Lat float64
	Lng float64
}

type Edge struct {
	DistanceM   int
	DurationSec int
}

type Provider interface {
	GetMatrix(ctx context.Context, points []Point) ([][]Edge, error)
}
