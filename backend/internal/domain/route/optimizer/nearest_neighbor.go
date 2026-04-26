package optimizer

import (
	"context"
	"math"
)

// NearestNeighborTW — эвристика ближайшего соседа с учётом временных окон (NNH-TW).
// Все временные величины хранятся в секундах Unix. Сложность — O(n³).
type NearestNeighborTW struct{}

func NewNearestNeighborTW() *NearestNeighborTW { return &NearestNeighborTW{} }

func (a *NearestNeighborTW) Name() string { return "nearest-neighbor-tw" }

// traversal — изменяемое состояние одного запуска оптимизации: где мы сейчас,
// какие узлы посещены, накопленные тайминги и метрики маршрута.
type traversal struct {
	g           *Graph
	cur         int
	currentTime int64
	visited     []bool
	order       []int
	timings     []StopTiming

	totalDistM      int
	totalTravelSec  int
	totalServiceSec int
	totalWaitSec    int
}

// appendNode добавляет узел next к маршруту и обновляет агрегаты + текущее время.
func (t *traversal) appendNode(next int) {
	e := t.g.Edges[t.cur][next]
	node := t.g.Nodes[next]

	arrivalSec := t.currentTime + int64(e.DurationSec)
	var waitSec int64
	if node.WindowStart >= 0 && arrivalSec < node.WindowStart {
		waitSec = node.WindowStart - arrivalSec
	}
	serviceStart := arrivalSec + waitSec
	serviceEnd := serviceStart + int64(node.DurationMin)*60

	t.totalDistM += e.DistanceM
	t.totalTravelSec += e.DurationSec
	t.totalServiceSec += node.DurationMin * 60
	t.totalWaitSec += int(waitSec)

	t.timings = append(t.timings, StopTiming{
		NodeIdx:           next,
		ArrivalSec:        arrivalSec,
		WaitSec:           int(waitSec),
		ServiceStartSec:   serviceStart,
		ServiceEndSec:     serviceEnd,
		TravelFromPrevSec: e.DurationSec,
	})
	t.visited[next] = true
	t.order = append(t.order, next)
	t.cur = next
	t.currentTime = serviceEnd
}

func (a *NearestNeighborTW) Optimize(_ context.Context, g *Graph, startTimeUnix int64, c Constraints) (Result, error) {
	n := len(g.Nodes)
	if n == 0 {
		return Result{}, nil
	}

	prereqs := buildPrereqs(n, c.PrecedencePairs)

	endIdx := -1
	if c.EndNodeIdx != nil {
		endIdx = *c.EndNodeIdx
	}

	cur := 0
	if c.StartNodeIdx != nil {
		cur = *c.StartNodeIdx
	} else {
		cur = startNode(g, prereqs, endIdx)
	}

	startServiceEnd := startTimeUnix + int64(g.Nodes[cur].DurationMin)*60
	t := &traversal{
		g:           g,
		cur:         cur,
		currentTime: startServiceEnd,
		visited:     make([]bool, n),
		order:       []int{cur},
		timings: []StopTiming{{
			NodeIdx:           cur,
			ArrivalSec:        startTimeUnix,
			WaitSec:           0,
			ServiceStartSec:   startTimeUnix,
			ServiceEndSec:     startServiceEnd,
			TravelFromPrevSec: 0,
		}},
		totalServiceSec: g.Nodes[cur].DurationMin * 60,
	}
	t.visited[cur] = true

	for len(t.order) < n {
		// Если остался только закреплённый конечный узел — ставим его и выходим.
		if endIdx >= 0 && !t.visited[endIdx] && onlyEndUnvisited(t.visited, endIdx) {
			t.appendNode(endIdx)
			break
		}

		next := pickNextFeasible(t, prereqs, endIdx)
		if next == -1 {
			next = pickNextFallback(t, prereqs, endIdx)
		}

		if next == -1 {
			// Узел не выбрать — возможно только при циклическом графе предшествования.
			if endIdx >= 0 && !t.visited[endIdx] {
				t.appendNode(endIdx)
			}
			break
		}

		t.appendNode(next)
	}

	return Result{
		Order:           t.order,
		Timings:         t.timings,
		TotalDistanceM:  t.totalDistM,
		TotalTravelSec:  t.totalTravelSec,
		TotalServiceSec: t.totalServiceSec,
		TotalWaitSec:    t.totalWaitSec,
	}, nil
}

// onlyEndUnvisited — true, если непосещён только закреплённый конечный узел.
func onlyEndUnvisited(visited []bool, endIdx int) bool {
	for i, v := range visited {
		if !v && i != endIdx {
			return false
		}
	}
	return true
}

// pickNextFeasible — проход 1: допустимый узел с наименьшим временем завершения
// + look-ahead. Кандидат "безопасен", если его посещение не делает недостижимым
// ни один другой узел с окном. Безопасные предпочитаются; внутри одного класса —
// меньшее время завершения, затем более срочный дедлайн.
func pickNextFeasible(t *traversal, prereqs [][]int, endIdx int) int {
	g := t.g
	next := -1
	var bestCompletion int64 = math.MaxInt64
	var bestDeadline int64 = math.MaxInt64
	bestSafe := false

	for i := range g.Nodes {
		if t.visited[i] || i == endIdx || !prereqsMet(i, t.visited, prereqs) {
			continue
		}
		arrival := t.currentTime + int64(g.Edges[t.cur][i].DurationSec)
		if !feasible(g.Nodes[i], arrival) {
			continue
		}

		comp := nodeCompletionTime(g.Nodes[i], arrival)
		deadline := g.Nodes[i].WindowEnd
		if deadline < 0 {
			deadline = math.MaxInt64
		}
		safe := !causesWindowMiss(i, comp, g, t.visited, endIdx)

		better := false
		switch {
		case safe && !bestSafe:
			better = true
		case safe == bestSafe:
			better = comp < bestCompletion ||
				(comp == bestCompletion && deadline < bestDeadline)
		}
		if better {
			bestCompletion = comp
			bestDeadline = deadline
			bestSafe = safe
			next = i
		}
	}
	return next
}

// pickNextFallback — проход 2: минимальное completion, окно игнорируется,
// предшествование сохраняется.
func pickNextFallback(t *traversal, prereqs [][]int, endIdx int) int {
	g := t.g
	next := -1
	var bestCompletion int64 = math.MaxInt64

	for i := range g.Nodes {
		if t.visited[i] || i == endIdx || !prereqsMet(i, t.visited, prereqs) {
			continue
		}
		arrival := t.currentTime + int64(g.Edges[t.cur][i].DurationSec)
		comp := nodeCompletionTime(g.Nodes[i], arrival)
		if comp < bestCompletion {
			bestCompletion = comp
			next = i
		}
	}
	return next
}

// buildPrereqs строит для каждого узла i список узлов, которые должны быть
// посещены до него согласно ограничениям PrecedencePairs.
func buildPrereqs(n int, pairs []PrecedencePair) [][]int {
	prereqs := make([][]int, n)
	for _, p := range pairs {
		prereqs[p.After] = append(prereqs[p.After], p.Before)
	}
	return prereqs
}

// prereqsMet возвращает true, если все узлы-предшественники узла i уже посещены.
func prereqsMet(i int, visited []bool, prereqs [][]int) bool {
	for _, pre := range prereqs[i] {
		if !visited[pre] {
			return false
		}
	}
	return true
}

// startNode выбирает стартовый узел среди допустимых (нет неудовлетворённых
// предшественников и узел не закреплён как конечный) по самому раннему окну.
func startNode(g *Graph, prereqs [][]int, endIdx int) int {
	eligible := func(i int) bool {
		if i == endIdx {
			return false
		}
		return len(prereqs[i]) == 0
	}

	best := -1
	for i, node := range g.Nodes {
		if !eligible(i) {
			continue
		}
		if best == -1 {
			best = i
			continue
		}
		// Предпочитаем узел с более ранним открытием окна; узел без окна проигрывает узлу с окном.
		bestHasWin := g.Nodes[best].WindowStart >= 0
		curHasWin := node.WindowStart >= 0
		switch {
		case curHasWin && !bestHasWin:
			best = i
		case curHasWin && bestHasWin && node.WindowStart < g.Nodes[best].WindowStart:
			best = i
		}
	}
	if best == -1 {
		return 0
	}
	return best
}

// causesWindowMiss — look-ahead на один шаг: возвращает true, если после завершения
// обслуживания в cand хотя бы один другой узел с окном становится недостижимым
// даже при прямом переезде cand→j.
func causesWindowMiss(cand int, afterCandSec int64, g *Graph, visited []bool, endIdx int) bool {
	for j := 0; j < len(g.Nodes); j++ {
		if visited[j] || j == cand || j == endIdx {
			continue
		}
		if g.Nodes[j].WindowEnd < 0 {
			continue
		}
		arrivalAtJ := afterCandSec + int64(g.Edges[cand][j].DurationSec)
		if !feasible(g.Nodes[j], arrivalAtJ) {
			return true
		}
	}
	return false
}

// nodeCompletionTime возвращает время завершения обслуживания в узле
// при прибытии в arrivalSec (с учётом возможного ожидания открытия окна).
func nodeCompletionTime(node Node, arrivalSec int64) int64 {
	start := arrivalSec
	if node.WindowStart >= 0 && start < node.WindowStart {
		start = node.WindowStart
	}
	return start + int64(node.DurationMin)*60
}

// feasible возвращает true, если обслуживание можно успеть до закрытия окна.
func feasible(node Node, arrivalSec int64) bool {
	if node.WindowEnd < 0 {
		return true
	}
	return arrivalSec <= node.WindowEnd-int64(node.DurationMin)*60
}
