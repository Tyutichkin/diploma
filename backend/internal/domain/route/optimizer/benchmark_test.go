package optimizer

import (
	"context"
	"math/rand"
	"testing"
)

func buildRandomGraph(n int, seed int64) *Graph {
	r := rand.New(rand.NewSource(seed))
	nodes := make([]Node, n)
	for i := 0; i < n; i++ {
		windowStart := int64(r.Intn(8*3600)) + 8*3600
		nodes[i] = Node{
			TaskID:      "",
			Lat:         55.75 + r.Float64()*0.2,
			Lng:         37.60 + r.Float64()*0.2,
			WindowStart: windowStart,
			WindowEnd:   windowStart + int64(3600+r.Intn(3*3600)),
			DurationMin: 10 + r.Intn(30),
		}
	}
	edges := make([][]Edge, n)
	for i := 0; i < n; i++ {
		edges[i] = make([]Edge, n)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			edges[i][j] = Edge{
				DistanceM:   1000 + r.Intn(5000),
				DurationSec: 120 + r.Intn(600),
			}
		}
	}
	return &Graph{Nodes: nodes, Edges: edges}
}

func benchmarkOptimize(b *testing.B, n int) {
	g := buildRandomGraph(n, 42)
	opt := NewNearestNeighborTW()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = opt.Optimize(ctx, g, 0, Constraints{})
	}
}

func BenchmarkOptimize_5(b *testing.B)  { benchmarkOptimize(b, 5) }
func BenchmarkOptimize_10(b *testing.B) { benchmarkOptimize(b, 10) }
func BenchmarkOptimize_20(b *testing.B) { benchmarkOptimize(b, 20) }
func BenchmarkOptimize_30(b *testing.B) { benchmarkOptimize(b, 30) }
func BenchmarkOptimize_50(b *testing.B) { benchmarkOptimize(b, 50) }
