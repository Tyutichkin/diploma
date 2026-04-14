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

	visited := make([]bool, n)
	order := make([]int, 0, n)

	var cur int
	if c.StartNodeIdx != nil {
		cur = *c.StartNodeIdx
	} else {
		cur = startNode(g, prereqs, endIdx)
	}
	visited[cur] = true
	order = append(order, cur)

	startServiceEnd := startTimeUnix + int64(g.Nodes[cur].DurationMin)*60
	timings := []StopTiming{{
		NodeIdx:           cur,
		ArrivalSec:        startTimeUnix,
		WaitSec:           0,
		ServiceStartSec:   startTimeUnix,
		ServiceEndSec:     startServiceEnd,
		TravelFromPrevSec: 0,
	}}

	currentTimeSec := startServiceEnd

	var totalDistM, totalTravelSec, totalServiceSec, totalWaitSec int
	totalServiceSec = g.Nodes[cur].DurationMin * 60

	for len(order) < n {
		// Если остался только закреплённый конечный узел — ставим его и выходим.
		if endIdx >= 0 && !visited[endIdx] {
			allOtherVisited := true
			for i := 0; i < n; i++ {
				if !visited[i] && i != endIdx {
					allOtherVisited = false
					break
				}
			}
			if allOtherVisited {
				var st StopTiming
				currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec, st =
					appendNode(endIdx, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
				timings = append(timings, st)
				visited[endIdx] = true
				order = append(order, endIdx)
				break
			}
		}

		next := -1
		var bestCompletionSec int64 = math.MaxInt64
		var bestDeadline int64 = math.MaxInt64
		bestSafe := false

		// Проход 1: допустимый узел с наименьшим временем завершения + look-ahead.
		// Кандидат "безопасен", если его посещение не делает недостижимым ни один
		// другой узел с окном. Безопасные всегда предпочитаются рискованным;
		// внутри одного класса — меньшее время завершения, затем более срочный дедлайн.
		for i := 0; i < n; i++ {
			if visited[i] || i == endIdx {
				continue
			}
			if !prereqsMet(i, visited, prereqs) {
				continue
			}
			e := g.Edges[cur][i]
			arrivalSec := currentTimeSec + int64(e.DurationSec)
			if !feasible(g.Nodes[i], arrivalSec) {
				continue
			}

			comp := nodeCompletionTime(g.Nodes[i], arrivalSec)
			deadline := g.Nodes[i].WindowEnd
			if deadline < 0 {
				deadline = math.MaxInt64
			}
			safe := !causesWindowMiss(i, comp, g, visited, endIdx)

			better := false
			switch {
			case safe && !bestSafe:
				better = true
			case safe == bestSafe:
				better = comp < bestCompletionSec ||
					(comp == bestCompletionSec && deadline < bestDeadline)
			}

			if better {
				bestCompletionSec = comp
				bestDeadline = deadline
				bestSafe = safe
				next = i
			}
		}

		// Проход 2: fallback — минимальное completion, окно игнорируется, предшествование сохраняется.
		if next == -1 {
			bestCompletionSec = math.MaxInt64
			for i := 0; i < n; i++ {
				if visited[i] || i == endIdx {
					continue
				}
				if !prereqsMet(i, visited, prereqs) {
					continue
				}
				e := g.Edges[cur][i]
				arrivalSec := currentTimeSec + int64(e.DurationSec)
				comp := nodeCompletionTime(g.Nodes[i], arrivalSec)
				if comp < bestCompletionSec {
					bestCompletionSec = comp
					next = i
				}
			}
		}

		if next == -1 {
			// Узел не выбрать — возможно только при циклическом графе предшествования.
			if endIdx >= 0 && !visited[endIdx] {
				var st StopTiming
				currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec, st =
					appendNode(endIdx, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
				timings = append(timings, st)
				visited[endIdx] = true
				order = append(order, endIdx)
			}
			break
		}

		var st StopTiming
		currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec, st =
			appendNode(next, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
		timings = append(timings, st)
		visited[next] = true
		order = append(order, next)
		cur = next
	}

	return Result{
		Order:           order,
		Timings:         timings,
		TotalDistanceM:  totalDistM,
		TotalTravelSec:  totalTravelSec,
		TotalServiceSec: totalServiceSec,
		TotalWaitSec:    totalWaitSec,
	}, nil
}

// appendNode добавляет узел next к текущему состоянию обхода и возвращает
// обновлённые значения currentTimeSec, агрегатов и StopTiming для этого узла.
func appendNode(next, cur int, currentTimeSec int64, g *Graph, distM, travelSec, serviceSec, waitSec int) (int64, int, int, int, int, StopTiming) {
	e := g.Edges[cur][next]
	node := g.Nodes[next]

	arrivalSec := currentTimeSec + int64(e.DurationSec)
	var waitSec2 int64
	if node.WindowStart >= 0 && arrivalSec < node.WindowStart {
		waitSec2 = node.WindowStart - arrivalSec
	}

	serviceStart := arrivalSec + waitSec2
	serviceEnd := serviceStart + int64(node.DurationMin)*60

	distM += e.DistanceM
	travelSec += e.DurationSec
	serviceSec += node.DurationMin * 60
	waitSec += int(waitSec2)

	st := StopTiming{
		NodeIdx:           next,
		ArrivalSec:        arrivalSec,
		WaitSec:           int(waitSec2),
		ServiceStartSec:   serviceStart,
		ServiceEndSec:     serviceEnd,
		TravelFromPrevSec: e.DurationSec,
	}

	return serviceEnd, distM, travelSec, serviceSec, waitSec, st
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
