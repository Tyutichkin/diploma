package optimizer

import (
	"context"
	"math"
)

const algorithmName = "nearest-neighbor-tw"

// NearestNeighborTW реализация Optimizer
type NearestNeighborTW struct{}

func NewNearestNeighborTW() *NearestNeighborTW { return &NearestNeighborTW{} }

func (a *NearestNeighborTW) Name() string { return algorithmName }

func (a *NearestNeighborTW) Optimize(_ context.Context, g *Graph, startTimeUnix int64, c Constraints) (Result, error) {
	n := len(g.Nodes)
	if n == 0 {
		return Result{}, nil
	}

	prereqs := buildPrereqs(n, c.PrecedencePairs)
	endIdx := optionalIdx(c.EndNodeIdx)
	startIdx := chooseStartNode(g, prereqs, endIdx, c.StartNodeIdx)

	plan := newRoutePlan(g, startIdx, startTimeUnix)

	for len(plan.order) < n {
		// Все оставшиеся узлы посещены кроме закреплённого «конечного» — ставим его и выходим.
		if endIdx >= 0 && !plan.visited[endIdx] && onlyEndUnvisited(plan.visited, endIdx) {
			plan.append(endIdx)
			break
		}

		next := pickBestFeasible(plan, prereqs, endIdx)
		if next == -1 {
			next = pickFallback(plan, prereqs, endIdx)
		}
		if next == -1 {
			// Тупик: возможно при циклическом графе предшествования. Замыкаем endIdx если он остался.
			if endIdx >= 0 && !plan.visited[endIdx] {
				plan.append(endIdx)
			}
			break
		}

		plan.append(next)
	}

	return plan.result(), nil
}

// routePlan хранит инкрементальное состояние построения маршрута:
// какие узлы уже добавлены, текущая позиция и время, накопленные метрики.
type routePlan struct {
	graph       *Graph
	current     int
	currentTime int64
	visited     []bool
	order       []int
	timings     []StopTiming

	totalDistanceM  int
	totalTravelSec  int
	totalServiceSec int
	totalWaitSec    int
}

func newRoutePlan(g *Graph, startIdx int, startTimeUnix int64) *routePlan {
	startServiceEnd := startTimeUnix + int64(g.Nodes[startIdx].DurationMin)*60
	p := &routePlan{
		graph:       g,
		current:     startIdx,
		currentTime: startServiceEnd,
		visited:     make([]bool, len(g.Nodes)),
		order:       []int{startIdx},
		timings: []StopTiming{{
			NodeIdx:         startIdx,
			ArrivalSec:      startTimeUnix,
			ServiceStartSec: startTimeUnix,
			ServiceEndSec:   startServiceEnd,
		}},
		totalServiceSec: g.Nodes[startIdx].DurationMin * 60,
	}
	p.visited[startIdx] = true
	return p
}

func (p *routePlan) append(next int) {
	edge := p.graph.Edges[p.current][next]
	node := p.graph.Nodes[next]

	arrivalSec := p.currentTime + int64(edge.DurationSec)
	waitSec := int64(0)
	if node.WindowStart >= 0 && arrivalSec < node.WindowStart {
		waitSec = node.WindowStart - arrivalSec
	}
	serviceStart := arrivalSec + waitSec
	serviceEnd := serviceStart + int64(node.DurationMin)*60

	p.totalDistanceM += edge.DistanceM
	p.totalTravelSec += edge.DurationSec
	p.totalServiceSec += node.DurationMin * 60
	p.totalWaitSec += int(waitSec)

	p.timings = append(p.timings, StopTiming{
		NodeIdx:           next,
		ArrivalSec:        arrivalSec,
		WaitSec:           int(waitSec),
		ServiceStartSec:   serviceStart,
		ServiceEndSec:     serviceEnd,
		TravelFromPrevSec: edge.DurationSec,
	})
	p.visited[next] = true
	p.order = append(p.order, next)
	p.current = next
	p.currentTime = serviceEnd
}

func (p *routePlan) result() Result {
	return Result{
		Order:           p.order,
		Timings:         p.timings,
		TotalDistanceM:  p.totalDistanceM,
		TotalTravelSec:  p.totalTravelSec,
		TotalServiceSec: p.totalServiceSec,
		TotalWaitSec:    p.totalWaitSec,
	}
}

// pickBestFeasible выбирает следующий узел, в окно которого ещё успеваем,
// предпочитая кандидатов, не блокирующих окна оставшихся узлов (safe-first),
// а среди равных — с самым ранним завершением и ближайшим дедлайном.
func pickBestFeasible(p *routePlan, prereqs [][]int, endIdx int) int {
	best := -1
	var bestCompletion int64 = math.MaxInt64
	var bestDeadline int64 = math.MaxInt64
	bestSafe := false

	for i := range p.graph.Nodes {
		if !isCandidate(p, i, endIdx, prereqs) {
			continue
		}
		arrival := p.arrivalAt(i)
		if !arrivesBeforeWindowClose(p.graph.Nodes[i], arrival) {
			continue
		}

		completion := completionTime(p.graph.Nodes[i], arrival)
		deadline := windowEndOrInfinity(p.graph.Nodes[i])
		safe := !blocksOthersWindows(p, i, completion, endIdx)

		if betterCandidate(safe, completion, deadline, bestSafe, bestCompletion, bestDeadline) {
			best = i
			bestCompletion = completion
			bestDeadline = deadline
			bestSafe = safe
		}
	}
	return best
}

// pickFallback запасной выбор, когда феасиблов не нашлось: берём ближайший
// по времени завершения узел, игнорируя временные окна.
func pickFallback(p *routePlan, prereqs [][]int, endIdx int) int {
	best := -1
	var bestCompletion int64 = math.MaxInt64

	for i := range p.graph.Nodes {
		if !isCandidate(p, i, endIdx, prereqs) {
			continue
		}
		completion := completionTime(p.graph.Nodes[i], p.arrivalAt(i))
		if completion < bestCompletion {
			best = i
			bestCompletion = completion
		}
	}
	return best
}

func isCandidate(p *routePlan, i, endIdx int, prereqs [][]int) bool {
	if p.visited[i] || i == endIdx {
		return false
	}
	return prereqsMet(i, p.visited, prereqs)
}

func (p *routePlan) arrivalAt(target int) int64 {
	return p.currentTime + int64(p.graph.Edges[p.current][target].DurationSec)
}

func betterCandidate(safe bool, completion, deadline int64, bestSafe bool, bestCompletion, bestDeadline int64) bool {
	if safe && !bestSafe {
		return true
	}
	if safe != bestSafe {
		return false
	}
	if completion != bestCompletion {
		return completion < bestCompletion
	}
	return deadline < bestDeadline
}

func buildPrereqs(n int, pairs []PrecedencePair) [][]int {
	prereqs := make([][]int, n)
	for _, p := range pairs {
		prereqs[p.After] = append(prereqs[p.After], p.Before)
	}
	return prereqs
}

func prereqsMet(i int, visited []bool, prereqs [][]int) bool {
	for _, pre := range prereqs[i] {
		if !visited[pre] {
			return false
		}
	}
	return true
}

func onlyEndUnvisited(visited []bool, endIdx int) bool {
	for i, v := range visited {
		if !v && i != endIdx {
			return false
		}
	}
	return true
}

// chooseStartNode возвращает узел, с которого начинается маршрут: явно заданный
// в Constraints, иначе — выбранный эвристикой selectDefaultStartNode.
func chooseStartNode(g *Graph, prereqs [][]int, endIdx int, override *int) int {
	if override != nil {
		return *override
	}
	return selectDefaultStartNode(g, prereqs, endIdx)
}

// selectDefaultStartNode выбирает стартовый узел без предшественников
// с самым ранним открытием окна (узлы без окна проигрывают узлам с окном).
func selectDefaultStartNode(g *Graph, prereqs [][]int, endIdx int) int {
	best := -1
	for i, node := range g.Nodes {
		if i == endIdx || len(prereqs[i]) != 0 {
			continue
		}
		if best == -1 || prefersAsStart(node, g.Nodes[best]) {
			best = i
		}
	}
	if best == -1 {
		return 0
	}
	return best
}

// prefersAsStart: узел с заданным окном предпочтительнее узла без окна;
// среди двух с окнами — тот, чьё окно открывается раньше.
func prefersAsStart(candidate, current Node) bool {
	candHasWindow := candidate.WindowStart >= 0
	curHasWindow := current.WindowStart >= 0
	if candHasWindow && !curHasWindow {
		return true
	}
	if !candHasWindow {
		return false
	}
	return candidate.WindowStart < current.WindowStart
}

// blocksOthersWindows: если поставить cand сейчас и обслужить его, успеем ли
// мы потом обслужить остальные неурегулированные узлы с временными окнами.
func blocksOthersWindows(p *routePlan, cand int, candCompletion int64, endIdx int) bool {
	for j := range p.graph.Nodes {
		if p.visited[j] || j == cand || j == endIdx {
			continue
		}
		if p.graph.Nodes[j].WindowEnd < 0 {
			continue
		}
		arrivalAtJ := candCompletion + int64(p.graph.Edges[cand][j].DurationSec)
		if !arrivesBeforeWindowClose(p.graph.Nodes[j], arrivalAtJ) {
			return true
		}
	}
	return false
}

func completionTime(node Node, arrivalSec int64) int64 {
	serviceStart := arrivalSec
	if node.WindowStart >= 0 && serviceStart < node.WindowStart {
		serviceStart = node.WindowStart
	}
	return serviceStart + int64(node.DurationMin)*60
}

// arrivesBeforeWindowClose: успеваем ли начать обслуживание не позже закрытия окна.
func arrivesBeforeWindowClose(node Node, arrivalSec int64) bool {
	if node.WindowEnd < 0 {
		return true
	}
	return arrivalSec <= node.WindowEnd-int64(node.DurationMin)*60
}

func windowEndOrInfinity(node Node) int64 {
	if node.WindowEnd < 0 {
		return math.MaxInt64
	}
	return node.WindowEnd
}

func optionalIdx(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}
