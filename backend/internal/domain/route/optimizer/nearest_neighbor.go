package optimizer

import (
	"context"
	"math"
)

// NearestNeighborTW implements the Nearest Neighbour Heuristic for the
// Travelling Salesman Problem with Time Windows (NNH-TW).
//
// Algorithm:
//  1. Choose the starting node: the one with the earliest time-window open
//     time, or node 0 if no time windows are defined.
//  2. Mark it visited; set current time = startTime + service duration.
//  3. Repeat until all nodes are visited:
//     a. Among unvisited nodes, find the one reachable within its time window
//        that has the minimum travel duration from the current node (Pass 1).
//     b. If no such node exists, pick the globally nearest unvisited node
//        regardless of its time window (Pass 2 — fallback).
//     c. Compute wait time if we arrive before the window opens.
//     d. Advance current time: arrival + wait + service duration.
//  4. Return the ordered node indices and aggregate statistics.
//
// Сложность: O(n²).
type NearestNeighborTW struct{}

func NewNearestNeighborTW() *NearestNeighborTW { return &NearestNeighborTW{} }

func (a *NearestNeighborTW) Name() string { return "nearest-neighbor-tw" }

func (a *NearestNeighborTW) Optimize(_ context.Context, g *Graph, startTimeMins int) (Result, error) {
	n := len(g.Nodes)
	if n == 0 {
		return Result{}, nil
	}

	visited := make([]bool, n)
	order := make([]int, 0, n)

	cur := startNode(g)
	visited[cur] = true
	order = append(order, cur)

	currentTimeMins := startTimeMins + g.Nodes[cur].DurationMin

	var totalDistM, totalTravelSec, totalServiceSec, totalWaitSec int
	totalServiceSec = g.Nodes[cur].DurationMin * 60

	for len(order) < n {
		next := -1
		nextDurSec := math.MaxInt

		// Pass 1: nearest feasible node (respects time window).
		for i := 0; i < n; i++ {
			if visited[i] {
				continue
			}
			e := g.Edges[cur][i]
			arrivalMins := currentTimeMins + e.DurationSec/60
			if feasible(g.Nodes[i], arrivalMins) && e.DurationSec < nextDurSec {
				nextDurSec = e.DurationSec
				next = i
			}
		}

		// Pass 2: fallback — nearest node regardless of time window.
		if next == -1 {
			nextDurSec = math.MaxInt
			for i := 0; i < n; i++ {
				if visited[i] {
					continue
				}
				if e := g.Edges[cur][i]; e.DurationSec < nextDurSec {
					nextDurSec = e.DurationSec
					next = i
				}
			}
		}

		if next == -1 {
			break
		}

		e := g.Edges[cur][next]
		node := g.Nodes[next]

		arrivalMins := currentTimeMins + e.DurationSec/60
		waitMins := 0
		if node.WindowStart >= 0 && arrivalMins < node.WindowStart {
			waitMins = node.WindowStart - arrivalMins
		}

		totalDistM += e.DistanceM
		totalTravelSec += e.DurationSec
		totalServiceSec += node.DurationMin * 60
		totalWaitSec += waitMins * 60

		currentTimeMins = arrivalMins + waitMins + node.DurationMin
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

// startNode selects the starting node: the one with the earliest time-window
// open time, or node 0 if no time windows are defined.
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

// feasible returns true when service at node can begin by arrivalMins
// (i.e. we can complete service before the window closes).
func feasible(node Node, arrivalMins int) bool {
	if node.WindowEnd < 0 {
		return true
	}
	return arrivalMins <= node.WindowEnd-node.DurationMin
}
