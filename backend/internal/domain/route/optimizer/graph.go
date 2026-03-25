package optimizer

// Node represents a task location in the routing graph.
type Node struct {
	TaskID string
	Lat    float64
	Lng    float64

	// WindowStart is the earliest allowed service-start time,
	// expressed as minutes since midnight.  -1 means no constraint.
	WindowStart int

	// WindowEnd is the latest allowed service-start time,
	// expressed as minutes since midnight.  -1 means no constraint.
	// The service must begin no later than WindowEnd-DurationMin.
	WindowEnd int

	DurationMin int // service duration at this stop, in minutes
}

// Edge holds the directed travel cost between two nodes.
type Edge struct {
	DistanceM   int // metres
	DurationSec int // seconds
}

// Graph is a complete directed weighted graph of task nodes.
// Edges[i][j] is the cost to travel from Nodes[i] to Nodes[j].
type Graph struct {
	Nodes []Node
	Edges [][]Edge
}
