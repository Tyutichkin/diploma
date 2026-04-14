package distance

import "context"

// Point — географическая координата.
type Point struct {
	Lat float64
	Lng float64
}

// Edge — стоимость переезда между двумя точками (метры, секунды).
type Edge struct {
	DistanceM   int
	DurationSec int
}

// Provider строит матрицу расстояний; Matrix[i][j] — стоимость points[i]→points[j],
// диагональ всегда нулевая.
type Provider interface {
	GetMatrix(ctx context.Context, points []Point) ([][]Edge, error)
}
