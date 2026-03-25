package distance

import "context"

// Point is a geographic coordinate.
type Point struct {
	Lat float64
	Lng float64
}

// Edge holds the travel cost between two points.
type Edge struct {
	DistanceM   int // travel distance in metres
	DurationSec int // travel duration in seconds
}

// Provider computes a distance/duration matrix between a set of points.
// Matrix[i][j] is the cost from points[i] to points[j].
// Diagonal values (i == j) are always zero.
type Provider interface {
	GetMatrix(ctx context.Context, points []Point) ([][]Edge, error)
}
