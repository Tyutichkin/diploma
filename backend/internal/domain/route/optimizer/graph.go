package optimizer

type Node struct {
	TaskID      string
	Lat         float64
	Lng         float64
	WindowStart int64 // хранится в unix секундах
	WindowEnd   int64 // храниртся в unix секундах
	DurationMin int
}

type Edge struct {
	DistanceM   int
	DurationSec int
}

type Graph struct {
	Nodes []Node
	Edges [][]Edge // стоимость ИЗ Nodes[i] в Nodes[j]
}
