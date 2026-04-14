package optimizer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// testRef — 2024-01-01T00:00:00Z, используется как базовая дата для тестов.
var testRef = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

// hhmm конвертирует часы и минуты в Unix-секунды, используя 2024-01-01 как базовую дату.
func hhmm(h, m int) int64 {
	return testRef + int64(h*3600+m*60)
}

func makeGraph(nodes []Node, edges [][]Edge) *Graph {
	return &Graph{Nodes: nodes, Edges: edges}
}

// twoNodeGraph builds a minimal 2-node, 2×2 edge graph.
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

// noConstraints — вспомогательная переменная для вызовов без ограничений.
var noConstraints = Constraints{}

// ── NearestNeighborTW.Optimize tests ─────────────────────────────────────────

// 4.1.1 2 узла без time windows — оба посещены
func TestOptimize_TwoNodesNoWindows(t *testing.T) {
	g := twoNodeGraph(30, 30, 600, 600)
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 2)
	// Both nodes visited exactly once
	visited := map[int]bool{}
	for _, idx := range res.Order {
		assert.False(t, visited[idx], "node %d visited twice", idx)
		visited[idx] = true
	}
}

// 4.1.2 3 узла — очевидно ближайший сосед
func TestOptimize_ThreeNodesNearestNeighbor(t *testing.T) {
	// Node 0 → Node 1 (100 sec) is clearly nearest; Node 0 → Node 2 (9000 sec)
	// Node 1 → Node 2 (100 sec)
	// Starting at node 0, expect order 0 → 1 → 2
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
	// Starting node (no windows) = node 0
	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 1, res.Order[1])
	assert.Equal(t, 2, res.Order[2])
}

// 4.1.3 Time window — раннее начало
func TestOptimize_TimeWindowEarliestFirst(t *testing.T) {
	// Node 0 window 10:00-12:00, Node 1 window 08:00-09:00
	// Should visit node 1 first (earlier window)
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
	res, err := a.Optimize(context.Background(), g, hhmm(8, 0), noConstraints) // start 08:00
	require.NoError(t, err)
	require.Len(t, res.Order, 2)
	// Node 1 has earlier window → should be visited first
	assert.Equal(t, 1, res.Order[0], "node with earliest window should be visited first")
}

// 4.1.4 Ожидание перед time window
func TestOptimize_WaitBeforeWindow(t *testing.T) {
	// Node 0: window [08:00, 09:00], DurationMin=30 → startNode picks node 0 (earliest)
	// Node 1: window [11:00, 12:00], DurationMin=0
	// Start 08:00. After node 0: currentTimeSec=08:00+30min=08:30.
	// Travel 0→1 = 10 sec → arrival=08:30. Window opens 11:00 → wait>0.
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
	assert.Greater(t, res.TotalWaitSec, 0, "should have wait time when arriving before window opens")
}

// 4.1.6 Агрегатная статистика
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

// 4.1.7 Один узел → n=1, start node selected, no loop iteration → order=[0]
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

// 4.1.8 startTimeUnix = 08:00
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

// 4.1.9 Все ноды с одинаковым весом — каждый посещён ровно один раз
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
		assert.Equal(t, 1, seen[i], "node %d should be visited exactly once", i)
	}
}

// 4.1.10 Недостижимые пары (99999 сек) — алгоритм всё равно посещает все ноды
func TestOptimize_UnreachablePairs(t *testing.T) {
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	// 0→2 is "unreachable" but algorithm should still visit via Pass 2
	edges := [][]Edge{
		{{}, {DistanceM: 100, DurationSec: 99999}, {DistanceM: 100, DurationSec: 99999}},
		{{DistanceM: 100, DurationSec: 99999}, {}, {DistanceM: 100, DurationSec: 600}},
		{{DistanceM: 100, DurationSec: 99999}, {DistanceM: 100, DurationSec: 600}, {}},
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 3, "all nodes must be visited even with unreachable pairs")
}

// 4.1.5 Fallback (Pass 2) — нода недостижима в time window
func TestOptimize_FallbackPass(t *testing.T) {
	// Node 0 has no window; Node 1 has a window that's already passed by the time we arrive
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
	// Start at 09:00 → we arrive at t1 at 10:00, which is after window end (08:10)
	// Pass 1 finds no feasible node → Pass 2 selects nearest regardless
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 2, "both nodes should still be visited via Pass 2 fallback")
}

// ── Empty graph ───────────────────────────────────────────────────────────────

func TestOptimize_EmptyGraph(t *testing.T) {
	g := &Graph{Nodes: []Node{}, Edges: [][]Edge{}}
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Order, 0)
}

// ── Constraints tests ─────────────────────────────────────────────────────────

// 4.3.1 Фиксированная стартовая точка
func TestOptimize_FixedStartNode(t *testing.T) {
	// Without constraint node 0 would be chosen (no windows).
	// With StartNodeIdx=2 the algorithm must begin at node 2.
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
	assert.Equal(t, 2, res.Order[0], "first node must be the pinned start")
}

// 4.3.2 Фиксированная конечная точка
func TestOptimize_FixedEndNode(t *testing.T) {
	// 3 nodes; without constraint the nearest-neighbour order from node 0 would be 0→1→2.
	// With EndNodeIdx=1 the algorithm must end at node 1, so expected order: 0→2→1.
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
	assert.Equal(t, 1, res.Order[len(res.Order)-1], "last node must be the pinned end")
}

// 4.3.3 Фиксированные начало и конец
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
	assert.Equal(t, 0, res.Order[0], "first node must be the pinned start")
	assert.Equal(t, 3, res.Order[3], "last node must be the pinned end")
}

// 4.3.4 Ограничение предшествования — A до B
func TestOptimize_PrecedenceConstraint(t *testing.T) {
	// 3 nodes; without constraint NNH picks nearest (node 1 from 0 is nearest).
	// With precedence node2 before node1: after visiting node0, node2 must come before node1.
	nodes := []Node{
		{TaskID: "t0", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t1", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 10, WindowStart: -1, WindowEnd: -1},
	}
	edges := [][]Edge{
		// node1 is nearest from node0 (100 sec), but precedence forces node2 first
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
	// Find positions
	pos := make(map[int]int, 3)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[2], pos[1], "node 2 must come before node 1 (precedence constraint)")
}

// 4.3.5 Все узлы посещены при наличии ограничений
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
	assert.Len(t, res.Order, n, "all nodes must be visited")
	assert.Equal(t, 0, res.Order[0])
	assert.Equal(t, 4, res.Order[n-1])
	pos := make(map[int]int, n)
	for i, idx := range res.Order {
		pos[idx] = i
	}
	assert.Less(t, pos[1], pos[3], "precedence: node 1 before node 3")
}

// ── feasible() tests ──────────────────────────────────────────────────────────

// 4.2.1 Нет ограничений (WindowEnd=-1) → true
func TestFeasible_NoConstraints(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: -1, WindowEnd: -1}
	assert.True(t, feasible(node, 0))
	assert.True(t, feasible(node, 999))
}

// 4.2.2 Прибытие в пределах окна → true
func TestFeasible_WithinWindow(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(18, 0)}
	// arrival at 10:00, window end = 18:00 - 30min → feasible
	assert.True(t, feasible(node, hhmm(10, 0)))
}

// 4.2.3 Прибытие до открытия окна → true (we wait)
func TestFeasible_ArrivalBeforeWindowStart(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	// arrival at 07:00, window opens 09:00
	// feasible checks arrivalSec <= WindowEnd - 30*60
	// 07:00 <= 09:30 → true
	assert.True(t, feasible(node, hhmm(7, 0)))
}

// 4.2.4 Прибытие после закрытия окна → false
func TestFeasible_ArrivalAfterWindowEnd(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	// arrival at 10:00, window closes at 10:00, need 30 min service → false
	assert.False(t, feasible(node, hhmm(10, 0)))
}

// 4.2.5 Прибытие прямо в окно
func TestFeasible_ArrivalAtDeadline(t *testing.T) {
	node := Node{DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 0)}
	// arrival at 09:30 = WindowEnd - DurationMin*60 → exactly at deadline → feasible
	assert.True(t, feasible(node, hhmm(9, 30)))
}

// ── startNode() tests ─────────────────────────────────────────────────────────

func TestStartNode_EarliestWindow(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{WindowStart: hhmm(10, 0)},
			{WindowStart: hhmm(8, 0)}, // earliest
			{WindowStart: hhmm(12, 0)},
		},
	}
	assert.Equal(t, 1, startNode(g))
}

func TestStartNode_NoWindows(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{WindowStart: -1},
			{WindowStart: -1},
		},
	}
	// all no-window → returns 0
	assert.Equal(t, 0, startNode(g))
}
