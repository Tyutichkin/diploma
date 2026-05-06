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

// Matrix[i][j] — стоимость points[i]→points[j], диагональ всегда нулевая.
type Provider interface {
	GetMatrix(ctx context.Context, points []Point) ([][]Edge, error)
}
