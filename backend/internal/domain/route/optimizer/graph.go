package optimizer

// Node — узел графа маршрутизации (точка задачи).
// WindowStart/WindowEnd хранятся в Unix-секундах; -1 означает отсутствие ограничения.
// Обслуживание должно начаться не позднее WindowEnd - DurationMin*60.
type Node struct {
	TaskID      string
	Lat         float64
	Lng         float64
	WindowStart int64
	WindowEnd   int64
	DurationMin int
}

// Edge — стоимость переезда между двумя узлами (метры, секунды).
type Edge struct {
	DistanceM   int
	DurationSec int
}

// Graph — полный ориентированный взвешенный граф. Edges[i][j] — стоимость Nodes[i]→Nodes[j].
type Graph struct {
	Nodes []Node
	Edges [][]Edge
}
