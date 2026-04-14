package optimizer

import (
	"context"
	"math"
)

// NearestNeighborTW implements the Nearest Neighbour Heuristic for the
// Travelling Salesman Problem with Time Windows (NNH-TW).
//
// Algorithm:
//  1. Choose the starting node: pinned via Constraints.StartNodeIdx, or the
//     node with the earliest time-window open time, or node 0 by default.
//  2. Mark it visited; set current time = startTime + service duration.
//  3. Repeat until all nodes are visited:
//     a. Among unvisited feasible nodes, find the one with the earliest
//        completion time (arrival + wait + service) that does NOT cause any
//        other unvisited time-windowed node to become unreachable (look-ahead).
//        If all candidates are "risky", pick the best completion time anyway.
//        Ties broken by deadline urgency. Precedence constraints are respected.
//     b. If no feasible node exists, pick the globally nearest unvisited node
//        regardless of its time window, still respecting precedence (Pass 2).
//     c. Compute wait time if we arrive before the window opens.
//     d. Advance current time: arrival + wait + service duration.
//  4. If an end node is pinned, append it last.
//  5. Return the ordered node indices and aggregate statistics.
//
// Все временны́е величины хранятся в секундах Unix.
// Сложность: O(n³) (look-ahead adds O(n) per candidate check).
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
		cur = startNode(g)
	}
	visited[cur] = true
	order = append(order, cur)

	// Timing for the start node: arrival = startTime, no travel, no wait.
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
		// If the only unvisited node is the pinned end node, place it and finish.
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
		bestSafe := false // whether the current best candidate is "safe" (no window miss)

		// Pass 1: feasible node with earliest completion time + one-step look-ahead.
		// A candidate is "safe" if visiting it does not make any other unvisited
		// time-windowed node unreachable.  Safe candidates are always preferred
		// over risky ones.  Within the same safety class, pick earliest completion
		// time; break ties by earliest deadline (most urgent).
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
				// Safe always beats risky.
				better = true
			case safe == bestSafe:
				// Same safety class: prefer earlier completion, then tighter deadline.
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

		// Pass 2: fallback — earliest completion regardless of time window (precedence still respected).
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
			// No selectable node: can only happen with a circular precedence graph.
			// Place the pinned end node if it remains, then stop.
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

// startNode selects the starting node: the one with the earliest time-window
// open time (Unix seconds), or node 0 if no time windows are defined.
func startNode(g *Graph) int {
	best := 0
	for i, node := range g.Nodes {
		if node.WindowStart < 0 {
			continue
		}
		if g.Nodes[best].WindowStart < 0 || node.WindowStart < g.Nodes[best].WindowStart {
			best = i
		}
	}
	return best
}

// causesWindowMiss returns true if, after finishing service at candidate node
// (whose completion time is afterCandSec), at least one other unvisited
// time-windowed node would become unreachable (infeasible in the best case —
// i.e. travelling directly from candidate to that node).
// This is the one-step look-ahead that prevents the greedy heuristic from
// "wasting time" on flexible tasks while urgent windows close.
func causesWindowMiss(cand int, afterCandSec int64, g *Graph, visited []bool, endIdx int) bool {
	for j := 0; j < len(g.Nodes); j++ {
		if visited[j] || j == cand || j == endIdx {
			continue
		}
		if g.Nodes[j].WindowEnd < 0 {
			continue // no deadline — cannot miss
		}
		arrivalAtJ := afterCandSec + int64(g.Edges[cand][j].DurationSec)
		if !feasible(g.Nodes[j], arrivalAtJ) {
			return true
		}
	}
	return false
}

// nodeCompletionTime returns the earliest time at which service at node would
// finish, given that the vehicle arrives at arrivalSec.  If a time window is
// defined the vehicle waits until WindowStart before beginning service.
func nodeCompletionTime(node Node, arrivalSec int64) int64 {
	start := arrivalSec
	if node.WindowStart >= 0 && start < node.WindowStart {
		start = node.WindowStart
	}
	return start + int64(node.DurationMin)*60
}

// feasible returns true when service at node can begin by arrivalSec
// (i.e. we can complete service before the window closes).
func feasible(node Node, arrivalSec int64) bool {
	if node.WindowEnd < 0 {
		return true
	}
	return arrivalSec <= node.WindowEnd-int64(node.DurationMin)*60
}
