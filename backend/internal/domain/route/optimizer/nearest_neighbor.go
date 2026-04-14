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
//     a. Among unvisited nodes (excluding any pinned end node), find the one
//        reachable within its time window that has the minimum travel duration
//        from the current node, respecting precedence constraints (Pass 1).
//     b. If no such node exists, pick the globally nearest unvisited node
//        regardless of its time window, still respecting precedence (Pass 2).
//     c. Compute wait time if we arrive before the window opens.
//     d. Advance current time: arrival + wait + service duration.
//  4. If an end node is pinned, append it last.
//  5. Return the ordered node indices and aggregate statistics.
//
// Все временны́е величины хранятся в секундах Unix.
// Сложность: O(n²).
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

	currentTimeSec := startTimeUnix + int64(g.Nodes[cur].DurationMin)*60

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
				currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec =
					appendNode(endIdx, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
				visited[endIdx] = true
				order = append(order, endIdx)
				break
			}
		}

		next := -1
		nextDurSec := math.MaxInt

		// Pass 1: nearest feasible node (respects time window and precedence).
		for i := 0; i < n; i++ {
			if visited[i] || i == endIdx {
				continue
			}
			if !prereqsMet(i, visited, prereqs) {
				continue
			}
			e := g.Edges[cur][i]
			arrivalSec := currentTimeSec + int64(e.DurationSec)
			if feasible(g.Nodes[i], arrivalSec) && e.DurationSec < nextDurSec {
				nextDurSec = e.DurationSec
				next = i
			}
		}

		// Pass 2: fallback — nearest node regardless of time window (precedence still respected).
		if next == -1 {
			nextDurSec = math.MaxInt
			for i := 0; i < n; i++ {
				if visited[i] || i == endIdx {
					continue
				}
				if !prereqsMet(i, visited, prereqs) {
					continue
				}
				if e := g.Edges[cur][i]; e.DurationSec < nextDurSec {
					nextDurSec = e.DurationSec
					next = i
				}
			}
		}

		if next == -1 {
			// No selectable node: can only happen with a circular precedence graph.
			// Place the pinned end node if it remains, then stop.
			if endIdx >= 0 && !visited[endIdx] {
				currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec =
					appendNode(endIdx, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
				visited[endIdx] = true
				order = append(order, endIdx)
			}
			break
		}

		currentTimeSec, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec =
			appendNode(next, cur, currentTimeSec, g, totalDistM, totalTravelSec, totalServiceSec, totalWaitSec)
		visited[next] = true
		order = append(order, next)
		cur = next
	}

	return Result{
		Order:           order,
		TotalDistanceM:  totalDistM,
		TotalTravelSec:  totalTravelSec,
		TotalServiceSec: totalServiceSec,
		TotalWaitSec:    totalWaitSec,
	}, nil
}

// appendNode добавляет узел next к текущему состоянию обхода и возвращает
// обновлённые значения currentTimeSec и агрегатов.
func appendNode(next, cur int, currentTimeSec int64, g *Graph, distM, travelSec, serviceSec, waitSec int) (int64, int, int, int, int) {
	e := g.Edges[cur][next]
	node := g.Nodes[next]

	arrivalSec := currentTimeSec + int64(e.DurationSec)
	var waitSec2 int64
	if node.WindowStart >= 0 && arrivalSec < node.WindowStart {
		waitSec2 = node.WindowStart - arrivalSec
	}

	distM += e.DistanceM
	travelSec += e.DurationSec
	serviceSec += node.DurationMin * 60
	waitSec += int(waitSec2)

	return arrivalSec + waitSec2 + int64(node.DurationMin)*60, distM, travelSec, serviceSec, waitSec
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

// feasible returns true when service at node can begin by arrivalSec
// (i.e. we can complete service before the window closes).
func feasible(node Node, arrivalSec int64) bool {
	if node.WindowEnd < 0 {
		return true
	}
	return arrivalSec <= node.WindowEnd-int64(node.DurationMin)*60
}
