package optimizer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRef — базовая дата тестов, 2024-01-01T00:00:00Z.
var testRef = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

func hhmm(h, m int) int64 {
	return testRef + int64(h*3600+m*60)
}

func makeGraph(nodes []Node, edges [][]Edge) *Graph {
	return &Graph{Nodes: nodes, Edges: edges}
}

func twoNodeGraph(dur0, dur1, travel01, travel10 int) *Graph {
	nodes := []Node{
		{TaskID: "task-0", DurationMin: dur0, WindowStart: -1, WindowEnd: -1},
		{TaskID: "task-1", DurationMin: dur1, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{DistanceM: 1000, DurationSec: travel01}, {DistanceM: 0, DurationSec: 0}},
		{{DistanceM: 1000, DurationSec: travel10}, {DistanceM: 0, DurationSec: 0}},
	}
	return &Graph{Nodes: nodes, Edges: edges}
}

var noConstraints = Constraints{}

// 4.1.1 два узла без окон — оба посещены.
func TestOptimize_TwoNodesNoWindows(t *testing.T) {
	g := twoNodeGraph(30, 30, 600, 600)
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 2)
	visited := map[int]bool{}
	for _, idx := range res.Order {
		assert.False(t, visited[idx], "node %d visited twice", idx)
		visited[idx] = true
	}
}

// 4.1.2 три узла — очевидный ближайший сосед (ожидаемый порядок 0→1→2).
func TestOptimize_ThreeNodesNearestNeighbor(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 500, DurationSec: 100}, {DistanceM: 9000, DurationSec: 9000}},
		{{DistanceM: 100, DurationSec: 100}, {}, {DistanceM: 500, DurationSec: 100}},
		{{DistanceM: 9000, DurationSec: 9000}, {DistanceM: 500, DurationSec: 100}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, 3)
	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 1, res.Order[1])
	assert.Equal(t, 2, res.Order[2])
}

// 4.1.3 узел с более ранним окном посещается первым.
func TestOptimize_TimeWindowEarliestFirst(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: hhmm(10, 0), WindowEnd: hhmm(12, 0)},
		{TaskID: "t1", DurationMin: 30, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 1000, DurationSec: 300}},
		{{DistanceM: 1000, DurationSec: 300}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, 2)
	assert.Equal(t, 1, res.Order[0], "узел с более ранним окном должен быть первым")
}

// 4.1.4 ожидание перед открытием окна.
func TestOptimize_WaitBeforeWindow(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
		{TaskID: "t1", DurationMin: 0, WindowStart: hhmm(11, 0), WindowEnd: hhmm(12, 0)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 100, DurationSec: 10}},
		{{DistanceM: 100, DurationSec: 10}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints)
	require.NoError(t, err)
	assert.Greater(t, res.TotalWaitSec, 0, "ожидается ненулевое ожидание при приезде до открытия окна")
}

// 4.1.6 агрегатная статистика.
func TestOptimize_AggregateStats(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 45, WindowStart: -1, WindowEnd: -1},
	}
	travelSec := 600
	distM := 5000
	edges := [][]Edge{
		{{}, {DistanceM: distM, DurationSec: travelSec}},
		{{DistanceM: distM, DurationSec: travelSec}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Equal(t, distM, res.TotalDistanceM)
	assert.Equal(t, travelSec, res.TotalTravelSec)
	expectedServiceSec := (30 + 45) * 60
	assert.Equal(t, expectedServiceSec, res.TotalServiceSec)
}

// 4.1.7 один узел.
func TestOptimize_OneNode(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 1)
}

// 4.1.8 startTimeUnix = 08:00.
func TestOptimize_StartTime0800(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 0, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
		{TaskID: "t1", DurationMin: 0, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 100, DurationSec: 60}},
		{{DistanceM: 100, DurationSec: 60}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 2)
}

// 4.1.9 все рёбра равны — каждый узел посещён ровно один раз.
func TestOptimize_SymmetricMatrix(t *testing.T) {
	n := 4
	nodes := make([]Node, n)
	for i := range nodes {
		nodes[i] = Node{TaskID: "t", DurationMin: 10, WindowStart: -1, WindowEnd: -1}
	}
	edges := make([][]Edge, n)
	for i := range edges {
		edges[i] = make([]Edge, n)
		for j := range edges[i] {
			if i != j {
				edges[i][j] = Edge{DistanceM: 1000, DurationSec: 600}
			}
		}
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, n)
	seen := map[int]int{}
	for _, idx := range res.Order {
		seen[idx]++
	}
	for i := 0; i < n; i++ {
		assert.Equal(t, 1, seen[i], "узел %d должен быть посещён ровно раз", i)
	}
}

// 4.1.10 недостижимые пары (99999 сек) — всё равно посещаются через Pass 2.
func TestOptimize_UnreachablePairs(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 100, DurationSec: 99999}, {DistanceM: 100, DurationSec: 99999}},
		{{DistanceM: 100, DurationSec: 99999}, {}, {DistanceM: 100, DurationSec: 600}},
		{{DistanceM: 100, DurationSec: 99999}, {DistanceM: 100, DurationSec: 600}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 3, "все узлы должны быть посещены даже при недостижимых парах")
}

// 4.1.5 fallback (Pass 2) — узел недостижим в окне.
func TestOptimize_FallbackPass(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: hhmm(8, 0), WindowEnd: hhmm(8, 10)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 1000, DurationSec: 3600}}, // 1 hour travel
		{{DistanceM: 1000, DurationSec: 3600}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 2, "оба узла должны быть посещены через Pass 2")
}

func TestOptimize_EmptyGraph(t *testing.T) {
	g := &Graph{Nodes: []Node{}, Edges: [][]Edge{}}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 0)
}

// 4.3.1 фиксированная стартовая точка.
func TestOptimize_FixedStartNode(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 100, DurationSec: 100}, {DistanceM: 100, DurationSec: 100}},
		{{DistanceM: 100, DurationSec: 100}, {}, {DistanceM: 100, DurationSec: 100}},
		{{DistanceM: 100, DurationSec: 100}, {DistanceM: 100, DurationSec: 100}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	startIdx := 2
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{StartNodeIdx: &startIdx})
	require.NoError(t, err)
	require.Len(t, res.Order, 3)
	assert.Equal(t, 2, res.Order[0], "первый узел должен совпадать с закреплённым стартом")
}

// 4.3.2 фиксированная конечная точка.
func TestOptimize_FixedEndNode(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 500, DurationSec: 100}, {DistanceM: 800, DurationSec: 200}},
		{{DistanceM: 500, DurationSec: 100}, {}, {DistanceM: 500, DurationSec: 100}},
		{{DistanceM: 800, DurationSec: 200}, {DistanceM: 500, DurationSec: 100}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	endIdx := 1
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{EndNodeIdx: &endIdx})
	require.NoError(t, err)
	require.Len(t, res.Order, 3)
	assert.Equal(t, 1, res.Order[len(res.Order)-1], "последний узел должен совпадать с закреплённым концом")
}

// 4.3.3 фиксированные начало и конец.
func TestOptimize_FixedStartAndEnd(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t3", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	sym := Edge{DistanceM: 100, DurationSec: 60}
	edges := [][]Edge{
		{{}, sym, sym, sym},
		{sym, {}, sym, sym},
		{sym, sym, {}, sym},
		{sym, sym, sym, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	startIdx, endIdx := 0, 3
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{StartNodeIdx: &startIdx, EndNodeIdx: &endIdx})
	require.NoError(t, err)
	require.Len(t, res.Order, 4)
	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 3, res.Order[3])
}

// 4.3.4 ограничение предшествования A→B.
func TestOptimize_PrecedenceConstraint(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		// без ограничений 0→1 ближе, но предшествование заставляет идти через 2.
		{{}, {DistanceM: 100, DurationSec: 100}, {DistanceM: 500, DurationSec: 500}},
		{{DistanceM: 100, DurationSec: 100}, {}, {DistanceM: 100, DurationSec: 100}},
		{{DistanceM: 500, DurationSec: 500}, {DistanceM: 100, DurationSec: 100}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{
		PrecedencePairs: []PrecedencePair{{Before: 2, After: 1}},
	})
	require.NoError(t, err)
	require.Len(t, res.Order, 3)
	pos := make(map[int]int, 3)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[2], pos[1], "узел 2 должен быть раньше узла 1 по предшествованию")
}

// 4.3.5 все узлы посещены при заданных ограничениях.
func TestOptimize_AllVisitedWithConstraints(t *testing.T) {
	n := 5
	nodes := make([]Node, n)
	for i := range nodes {
		nodes[i] = Node{TaskID: "t", DurationMin: 5, WindowStart: -1, WindowEnd: -1}
	}
	edges := make([][]Edge, n)
	for i := range edges {
		edges[i] = make([]Edge, n)
		for j := range edges[i] {
			if i != j {
				edges[i][j] = Edge{DistanceM: 100, DurationSec: 60}
			}
		}
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	startIdx, endIdx := 0, 4
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{
		StartNodeIdx:    &startIdx,
		EndNodeIdx:      &endIdx,
		PrecedencePairs: []PrecedencePair{{Before: 1, After: 3}},
	})
	require.NoError(t, err)
	assert.Len(t, res.Order, n)
	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 4, res.Order[n-1])
	pos := make(map[int]int, n)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[1], pos[3])
}

// Регрессия: близкий узел с поздним окном (14:00-16:00) выбирался раньше узла
// с ранним окном (11:00-12:00), потому что алгоритм смотрел только на travel time.
func TestOptimize_NearbyLateWindowNotPreferred(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 30)},
		{TaskID: "t1", DurationMin: 20, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 45, WindowStart: hhmm(11, 0), WindowEnd: hhmm(12, 0)},
		{TaskID: "t3", DurationMin: 20, WindowStart: hhmm(14, 0), WindowEnd: hhmm(16, 0)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 3000, DurationSec: 300}, {DistanceM: 5000, DurationSec: 600}, {DistanceM: 500, DurationSec: 60}},
		{{DistanceM: 3000, DurationSec: 300}, {}, {DistanceM: 3000, DurationSec: 300}, {DistanceM: 3000, DurationSec: 300}},
		{{DistanceM: 5000, DurationSec: 600}, {DistanceM: 3000, DurationSec: 300}, {}, {DistanceM: 3000, DurationSec: 300}},
		{{DistanceM: 500, DurationSec: 60}, {DistanceM: 3000, DurationSec: 300}, {DistanceM: 3000, DurationSec: 300}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, 4)

	pos := make(map[int]int, 4)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[2], pos[3],
		"узел с окном 11:00-12:00 должен идти раньше узла с окном 14:00-16:00")
}

// Узел без окна должен заполнять паузу, а не вызывать простой перед поздним окном.
func TestOptimize_NoWindowTaskFillsGap(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: hhmm(10, 0), WindowEnd: hhmm(11, 0)},
	}
	sym := Edge{DistanceM: 2000, DurationSec: 300}
	edges := [][]Edge{
		{{}, sym, sym},
		{sym, {}, sym},
		{sym, sym, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, 3)

	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 1, res.Order[1], "задача без окна заполняет паузу")
	assert.Equal(t, 2, res.Order[2])
}

// Несколько задач с окнами — порядок должен уважать хронологию окон.
func TestOptimize_MultipleWindowsChronologicalOrder(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 20, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)},
		{TaskID: "t1", DurationMin: 20, WindowStart: hhmm(10, 30), WindowEnd: hhmm(11, 30)},
		{TaskID: "t2", DurationMin: 20, WindowStart: hhmm(12, 0), WindowEnd: hhmm(13, 0)},
		{TaskID: "t3", DurationMin: 20, WindowStart: hhmm(13, 30), WindowEnd: hhmm(14, 30)},
	}
	sym := Edge{DistanceM: 1000, DurationSec: 300}
	edges := make([][]Edge, 4)
	for i := range edges {
		edges[i] = make([]Edge, 4)
		for j := range edges[i] {
			if i != j {
				edges[i][j] = sym
			}
		}
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, 4)
	assert.Equal(t, []int{0, 1, 2, 3}, res.Order)
}

// При равном completion-time приоритет у узла с более жёстким дедлайном.
func TestOptimize_TightDeadlineTiebreaker(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)},
		{TaskID: "t2", DurationMin: 10, WindowStart: hhmm(9, 0), WindowEnd: hhmm(18, 0)},
	}
	sym := Edge{DistanceM: 2000, DurationSec: 300}
	edges := [][]Edge{
		{{}, sym, sym},
		{sym, {}, sym},
		{sym, sym, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	startIdx := 0
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{StartNodeIdx: &startIdx})
	require.NoError(t, err)
	require.Len(t, res.Order, 3)

	pos := make(map[int]int, 3)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[1], pos[2])
}

// Выбор гибкого узла первым не должен приводить к пропуску жёсткого дедлайна.
func TestOptimize_FlexibleFirstDoesNotMissTightDeadline(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: hhmm(9, 20), WindowEnd: hhmm(9, 40)},
		{TaskID: "t2", DurationMin: 10, WindowStart: hhmm(9, 0), WindowEnd: hhmm(18, 0)},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 2000, DurationSec: 300}, {DistanceM: 1000, DurationSec: 120}},
		{{DistanceM: 2000, DurationSec: 300}, {}, {DistanceM: 2000, DurationSec: 300}},
		{{DistanceM: 1000, DurationSec: 120}, {DistanceM: 2000, DurationSec: 300}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	startIdx := 0
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), Constraints{StartNodeIdx: &startIdx})
	require.NoError(t, err)
	require.Len(t, res.Order, 3)
	seen := map[int]bool{}
	for _, idx := range res.Order {
		seen[idx] = true
	}
	for i := 0; i < 3; i++ {
		assert.True(t, seen[i], "узел %d должен быть посещён", i)
	}
}

// Look-ahead не даёт пропустить жёсткое окно, пока заполняются задачи без окон.
func TestOptimize_LookAheadPreventsWindowMiss(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 30)},
		{TaskID: "t1", DurationMin: 20, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 45, WindowStart: hhmm(11, 0), WindowEnd: hhmm(12, 0)},
		{TaskID: "t3", DurationMin: 30, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t4", DurationMin: 20, WindowStart: -1, WindowEnd: -1},
	}
	travel := Edge{DistanceM: 5000, DurationSec: 600}
	n := len(nodes)
	edges := make([][]Edge, n)
	for i := range edges {
		edges[i] = make([]Edge, n)
		for j := range edges[i] {
			if i != j {
				edges[i][j] = travel
			}
		}
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Order, n)

	pos := make(map[int]int, n)
	for i, idx := range res.Order {
		pos[idx] = i
	}

	// Проигрываем маршрут и проверяем, что узел 2 достигнут до дедлайна.
	timeSec := hhmm(9, 0)
	prev := res.Order[0]
	timeSec += int64(g.Nodes[prev].DurationMin) * 60
	for step := 1; step < n; step++ {
		cur := res.Order[step]
		timeSec += int64(g.Edges[prev][cur].DurationSec)
		if g.Nodes[cur].WindowStart >= 0 && timeSec < g.Nodes[cur].WindowStart {
			timeSec = g.Nodes[cur].WindowStart
		}
		if cur == 2 {
			deadline := g.Nodes[2].WindowEnd - int64(g.Nodes[2].DurationMin)*60
			assert.LessOrEqual(t, timeSec, deadline,
				"узел 2 должен быть достигнут до дедлайна; arrived=%d, deadline=%d",
				timeSec-hhmm(0, 0), deadline-hhmm(0, 0))
			break
		}
		timeSec += int64(g.Nodes[cur].DurationMin) * 60
		prev = cur
	}
}

func TestNodeCompletionTime(t *testing.T) {
	n1 := Node{DurationMin: 30, WindowStart: -1, WindowEnd: -1}
	assert.Equal(t, hhmm(9, 30), nodeCompletionTime(n1, hhmm(9, 0)))

	n2 := Node{DurationMin: 20, WindowStart: hhmm(10, 0), WindowEnd: hhmm(12, 0)}
	assert.Equal(t, hhmm(10, 20), nodeCompletionTime(n2, hhmm(9, 0)))
	assert.Equal(t, hhmm(11, 20), nodeCompletionTime(n2, hhmm(11, 0)))
}

func TestOptimize_TimingsLength(t *testing.T) {
	g := twoNodeGraph(30, 30, 600, 600)
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Timings, len(res.Order))
}

func TestOptimize_TimingsFirstStop(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 20, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		{{}, {DistanceM: 1000, DurationSec: 300}},
		{{DistanceM: 1000, DurationSec: 300}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Timings, 2)

	first := res.Timings[0]
	assert.Equal(t, 0, first.TravelFromPrevSec)
	assert.Equal(t, hhmm(9, 0), first.ArrivalSec)
	assert.Equal(t, hhmm(9, 0), first.ServiceStartSec)
	assert.Equal(t, hhmm(9, 20), first.ServiceEndSec)
	assert.Equal(t, 0, first.WaitSec)
}

// Тайминги для графа из 3 узлов с ожиданием.
func TestOptimize_TimingsWithWait(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: hhmm(8, 0), WindowEnd: hhmm(9, 0)},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: hhmm(10, 0), WindowEnd: hhmm(11, 0)},
	}
	sym := Edge{DistanceM: 2000, DurationSec: 300}
	edges := [][]Edge{
		{{}, sym, sym},
		{sym, {}, sym},
		{sym, sym, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints)
	require.NoError(t, err)
	require.Len(t, res.Timings, 3)

	assert.Equal(t, []int{0, 1, 2}, res.Order)

	assert.Equal(t, hhmm(8, 0), res.Timings[0].ArrivalSec)
	assert.Equal(t, hhmm(8, 10), res.Timings[0].ServiceEndSec)

	assert.Equal(t, 300, res.Timings[1].TravelFromPrevSec)
	assert.Equal(t, hhmm(8, 15), res.Timings[1].ArrivalSec)
	assert.Equal(t, hhmm(8, 25), res.Timings[1].ServiceEndSec)
	assert.Equal(t, 0, res.Timings[1].WaitSec)

	assert.Equal(t, 300, res.Timings[2].TravelFromPrevSec)
	assert.Equal(t, hhmm(8, 30), res.Timings[2].ArrivalSec)
	assert.Equal(t, hhmm(10, 0), res.Timings[2].ServiceStartSec)
	assert.Equal(t, hhmm(10, 10), res.Timings[2].ServiceEndSec)
	assert.Equal(t, 90*60, res.Timings[2].WaitSec)
}

func TestOptimize_TimingsServiceDuration(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 45, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 15, WindowStart: -1, WindowEnd: -1},
	}
	sym := Edge{DistanceM: 1000, DurationSec: 600}
	edges := make([][]Edge, 3)
	for i := range edges {
		edges[i] = make([]Edge, 3)
		for j := range edges[i] {
			if i != j {
				edges[i][j] = sym
			}
		}
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	for i, st := range res.Timings {
		nodeIdx := res.Order[i]
		expectedDur := int64(g.Nodes[nodeIdx].DurationMin) * 60
		assert.Equal(t, expectedDur, st.ServiceEndSec-st.ServiceStartSec,
			"stop %d: длительность обслуживания не совпадает", i)
	}
}

// 4.2.1 нет ограничений → true.
func TestFeasible_NoConstraints(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: -1, WindowEnd: -1}
	assert.True(t, feasible(node, 0))
	assert.True(t, feasible(node, 999))
}

// 4.2.2 прибытие в пределах окна.
func TestFeasible_WithinWindow(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(18, 0)}
	assert.True(t, feasible(node, hhmm(10, 0)))
}

// 4.2.3 прибытие до открытия окна (ожидание допустимо).
func TestFeasible_ArrivalBeforeWindowStart(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	assert.True(t, feasible(node, hhmm(7, 0)))
}

// 4.2.4 прибытие после закрытия окна.
func TestFeasible_ArrivalAfterWindowEnd(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	assert.False(t, feasible(node, hhmm(10, 0)))
}

// 4.2.5 прибытие ровно в дедлайн (WindowEnd - DurationMin*60).
func TestFeasible_ArrivalAtDeadline(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	assert.True(t, feasible(node, hhmm(9, 30)))
}

func TestStartNode_EarliestWindow(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{WindowStart: hhmm(10, 0)},
			{WindowStart: hhmm(8, 0)}, // earliest
			{WindowStart: hhmm(12, 0)},
		},
	}
	assert.Equal(t, 1, startNode(g, make([][]int, len(g.Nodes)), -1))
}

func TestStartNode_NoWindows(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{WindowStart: -1},
			{WindowStart: -1},
		},
	}
	assert.Equal(t, 0, startNode(g, make([][]int, len(g.Nodes)), -1))
}

// Регрессия: узел-After не должен выбираться стартовым, даже если у него самое раннее окно.
func TestStartNode_RespectsPrecedence(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{WindowStart: hhmm(10, 0)},
			{WindowStart: hhmm(9, 30)},
			{WindowStart: hhmm(12, 0)},
		},
	}
	prereqs := buildPrereqs(len(g.Nodes), []PrecedencePair{{Before: 0, After: 1}})
	assert.Equal(t, 0, startNode(g, prereqs, -1))
}
