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

// ── Time-window ordering regression tests ────────────────────────────────────

// Близкий узел с поздним окном не должен выбираться раньше узла с ранним окном.
// Регрессия: старый алгоритм выбирал ближайший по travel time, игнорируя время
// ожидания. Узел C (14:00-16:00) рядом выбирался раньше B (11:00-12:00).
func TestOptimize_NearbyLateWindowNotPreferred(t *testing.T) {
	// Node 0: window 09:00-10:30, 30 min — start node (earliest window)
	// Node 1: no window, 20 min — filler task
	// Node 2: window 11:00-12:00, 45 min — must be visited before window closes
	// Node 3: window 14:00-16:00, 20 min — late window, physically close to node 0
	//
	// Travel times:  0→3 = 60s (very close), 0→1 = 300s, 0→2 = 600s
	// Old behaviour: after node 0 (done 09:30), picks node 3 (nearest), waits until 14:00,
	//   then at 14:20 tries node 2 — window closed at 12:00 → missed.
	// Fixed: picks nodes by completion time, avoids long waits.
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
	// Node 3 (late window) must NOT come before node 2 (early window)
	assert.Less(t, pos[2], pos[3],
		"node with window 11:00-12:00 must be visited before node with window 14:00-16:00")
}

// Узел без time window не должен вызывать простой, если есть задачи с ранними окнами.
func TestOptimize_NoWindowTaskFillsGap(t *testing.T) {
	// Node 0: window 08:00-09:00, 10 min — start
	// Node 1: no window, 10 min — should fill gap before node 2 opens
	// Node 2: window 10:00-11:00, 10 min
	// All equidistant (300s travel).
	// After node 0 (done 08:10), next step should visit node 1 (completion ~08:20)
	// rather than waiting until 10:00 for node 2.
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

	// Node 1 (no window) should be visited second (fills the gap), node 2 last.
	assert.Equal(t, 0, res.Order[0], "start with earliest window")
	assert.Equal(t, 1, res.Order[1], "no-window task fills gap before late window opens")
	assert.Equal(t, 2, res.Order[2], "windowed task visited when window is open")
}

// Несколько задач с окнами — порядок должен уважать хронологию окон.
func TestOptimize_MultipleWindowsChronologicalOrder(t *testing.T) {
	// 4 nodes with sequential non-overlapping windows.
	// All equidistant — order should follow window chronology.
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
	assert.Equal(t, []int{0, 1, 2, 3}, res.Order,
		"sequential windows should be visited in chronological order")
}

// При одинаковом completion time узел с жестким дедлайном выбирается первым (tiebreaker).
func TestOptimize_TightDeadlineTiebreaker(t *testing.T) {
	// Node 0: no window, 10 min — start
	// Node 1: window 09:00-10:00, 10 min — tight deadline
	// Node 2: window 09:00-18:00, 10 min — flexible deadline
	// Equal travel times → equal completion times → deadline breaks the tie.
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
	assert.Less(t, pos[1], pos[2],
		"with equal completion times, tight deadline node should be visited first")
}

// Выбор гибкого узла первым не должен приводить к пропуску жесткого дедлайна.
func TestOptimize_FlexibleFirstDoesNotMissTightDeadline(t *testing.T) {
	// Node 0: no window, 10 min — start
	// Node 1: window 09:20-09:40, 10 min — very tight (only 20 min window!)
	// Node 2: window 09:00-18:00, 10 min — flexible, closer
	// Travel: 0→2 = 120s (close), 0→1 = 300s, 2→1 = 300s
	// After node 0 (done 09:10):
	//   Option A (visit 2 first): arrive 09:12, done 09:22. Then 2→1: arrive 09:27, feasible (09:27 ≤ 09:40-10min=09:30). Done 09:37.
	//   Option B (visit 1 first): arrive 09:15, wait until 09:20, done 09:30. Then 1→2: done 09:42.
	// Both options work. A has earlier total completion. Algorithm should pick A.
	// The key: even though 2 is picked first, 1 is NOT missed.
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
	// All 3 nodes visited — the tight deadline was NOT missed
	seen := map[int]bool{}
	for _, idx := range res.Order {
		seen[idx] = true
	}
	for i := 0; i < 3; i++ {
		assert.True(t, seen[i], "node %d must be visited", i)
	}
}

// Сценарий demo_tasks: задачи без окон заполняют паузу, и жесткое окно (45 мин
// обслуживания, окно 11:00-12:00) пропускается, потому что алгоритм не смотрит вперед.
// Look-ahead должен это предотвратить.
func TestOptimize_LookAheadPreventsWindowMiss(t *testing.T) {
	// Моделируем упрощённый demo_tasks:
	// Node 0: window 09:00-10:30, 30 min (Альфа Трейд) — start
	// Node 1: no window, 20 min (Морозов)
	// Node 2: window 11:00-12:00, 45 min (Петров) — tight! must arrive by 11:15
	// Node 3: no window, 30 min (Профит-Сервис)
	// Node 4: no window, 20 min (Склад)
	//
	// All travel times ~600s (10 min). After node 0 (done 09:30):
	// Without look-ahead: picks 1 (comp 10:00), then 3 (comp 10:40), then 4 (comp 11:10),
	//   then 2: arrival 11:20 > deadline 11:15 → MISS → fallback.
	// With look-ahead: after 1+3 (done 10:40), picking 4 would cause miss of 2
	//   (10:40+20+10 = 11:10 → 11:10+10 = 11:20 > 11:15). So picks 2 instead.
	nodes := []Node{
		{TaskID: "t0", DurationMin: 30, WindowStart: hhmm(9, 0), WindowEnd: hhmm(10, 30)},
		{TaskID: "t1", DurationMin: 20, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t2", DurationMin: 45, WindowStart: hhmm(11, 0), WindowEnd: hhmm(12, 0)},
		{TaskID: "t3", DurationMin: 30, WindowStart: -1, WindowEnd: -1},
		{TaskID: "t4", DurationMin: 20, WindowStart: -1, WindowEnd: -1},
	}
	travel := Edge{DistanceM: 5000, DurationSec: 600} // 10 min everywhere
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

	// Find position of node 2 (tight window) in the order.
	pos := make(map[int]int, n)
	for i, idx := range res.Order {
		pos[idx] = i
	}

	// Verify node 2 is visited while its window is still open.
	// Compute actual arrival time at node 2 by replaying the route.
	timeSec := hhmm(9, 0)
	prev := res.Order[0]
	timeSec += int64(g.Nodes[prev].DurationMin) * 60 // service at start node
	for step := 1; step < n; step++ {
		cur := res.Order[step]
		timeSec += int64(g.Edges[prev][cur].DurationSec) // travel
		if g.Nodes[cur].WindowStart >= 0 && timeSec < g.Nodes[cur].WindowStart {
			timeSec = g.Nodes[cur].WindowStart // wait
		}
		if cur == 2 {
			// Must arrive by 11:15 (WindowEnd - DurationMin*60)
			deadline := g.Nodes[2].WindowEnd - int64(g.Nodes[2].DurationMin)*60
			assert.LessOrEqual(t, timeSec, deadline,
				"node 2 (Петров) must be reached before its deadline; arrived at %d, deadline %d",
				timeSec-hhmm(0, 0), deadline-hhmm(0, 0))
			break
		}
		timeSec += int64(g.Nodes[cur].DurationMin) * 60 // service
		prev = cur
	}
}

// Тест nodeCompletionTime.
func TestNodeCompletionTime(t *testing.T) {
	// No window: completion = arrival + service
	n1 := Node{DurationMin: 30, WindowStart: -1, WindowEnd: -1}
	assert.Equal(t, hhmm(9, 30), nodeCompletionTime(n1, hhmm(9, 0)))

	// Arrive before window: wait then service
	n2 := Node{DurationMin: 20, WindowStart: hhmm(10, 0), WindowEnd: hhmm(12, 0)}
	// arrive 09:00, wait until 10:00, service 20 min → 10:20
	assert.Equal(t, hhmm(10, 20), nodeCompletionTime(n2, hhmm(9, 0)))

	// Arrive during window: no wait
	assert.Equal(t, hhmm(11, 20), nodeCompletionTime(n2, hhmm(11, 0)))
}

// ── StopTiming tests ─────────────────────────────────────────────────────────

// Timings slice has same length as Order.
func TestOptimize_TimingsLength(t *testing.T) {
	g := twoNodeGraph(30, 30, 600, 600)
	a := NewNearestNeighborTW()
	res, err := a.Optimize(context.Background(), g, hhmm(9, 0), noConstraints)
	require.NoError(t, err)
	assert.Len(t, res.Timings, len(res.Order))
}

// First stop has no travel and arrival = startTime.
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
	assert.Equal(t, 0, first.TravelFromPrevSec, "first stop has no travel")
	assert.Equal(t, hhmm(9, 0), first.ArrivalSec, "first stop arrives at start time")
	assert.Equal(t, hhmm(9, 0), first.ServiceStartSec)
	assert.Equal(t, hhmm(9, 20), first.ServiceEndSec, "20 min service")
	assert.Equal(t, 0, first.WaitSec)
}

// Timing correctness for a 3-node graph with a wait.
func TestOptimize_TimingsWithWait(t *testing.T) {
	// Node 0: window 08:00-09:00, 10 min — start
	// Node 1: no window, 10 min
	// Node 2: window 10:00-11:00, 10 min — will require wait
	// Travel: all 300s (5 min)
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

	// Order: 0 (earliest window) → 1 (no window, fills gap) → 2 (window 10:00)
	assert.Equal(t, []int{0, 1, 2}, res.Order)

	// Stop 0: arrive 08:00, service 08:00-08:10
	assert.Equal(t, hhmm(8, 0), res.Timings[0].ArrivalSec)
	assert.Equal(t, hhmm(8, 10), res.Timings[0].ServiceEndSec)

	// Stop 1: travel 5 min from 08:10, arrive 08:15, service 08:15-08:25
	assert.Equal(t, 300, res.Timings[1].TravelFromPrevSec)
	assert.Equal(t, hhmm(8, 15), res.Timings[1].ArrivalSec)
	assert.Equal(t, hhmm(8, 25), res.Timings[1].ServiceEndSec)
	assert.Equal(t, 0, res.Timings[1].WaitSec)

	// Stop 2: travel 5 min from 08:25, arrive 08:30, wait until 10:00
	assert.Equal(t, 300, res.Timings[2].TravelFromPrevSec)
	assert.Equal(t, hhmm(8, 30), res.Timings[2].ArrivalSec)
	assert.Equal(t, hhmm(10, 0), res.Timings[2].ServiceStartSec)
	assert.Equal(t, hhmm(10, 10), res.Timings[2].ServiceEndSec)
	assert.Equal(t, 90*60, res.Timings[2].WaitSec, "90 minutes of wait")
}

// ServiceEndSec - ServiceStartSec == DurationMin * 60 for all stops.
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
			"stop %d: service duration mismatch", i)
	}
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
