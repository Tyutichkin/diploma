package optimizer

import "context"

// StopTiming — тайминги одной остановки маршрута.
type StopTiming struct {
	NodeIdx           int
	ArrivalSec        int64
	WaitSec           int
	ServiceStartSec   int64
	ServiceEndSec     int64
	TravelFromPrevSec int
}

// Result — результат запуска оптимизатора.
type Result struct {
	Order   []int
	Timings []StopTiming

	TotalDistanceM  int
	TotalTravelSec  int
	TotalServiceSec int
	TotalWaitSec    int
}

// PrecedencePair задаёт ограничение порядка: узел Before должен быть посещён
// строго до узла After. Оба значения — индексы в Graph.Nodes.
type PrecedencePair struct {
	Before int
	After  int
}

// Constraints содержит дополнительные ограничения на порядок обхода узлов.
// Нулевое значение означает отсутствие ограничений.
type Constraints struct {
	// StartNodeIdx фиксирует первый узел маршрута.
	// Если nil — стартовый узел выбирается алгоритмом.
	StartNodeIdx *int

	// EndNodeIdx фиксирует последний узел маршрута.
	// Если nil — конечный узел не фиксируется.
	EndNodeIdx *int

	// PrecedencePairs задаёт попарные ограничения порядка посещения.
	PrecedencePairs []PrecedencePair
}

// Optimizer ищет оптимальный порядок обхода узлов графа.
// Реализации должны быть безопасны для конкурентного использования.
type Optimizer interface {
	// Name — короткий идентификатор алгоритма, сохраняется в БД рядом с маршрутом.
	Name() string

	// Optimize возвращает индексы узлов в оптимальном порядке обхода.
	// startTimeUnix — время выезда в секундах Unix.
	Optimize(ctx context.Context, g *Graph, startTimeUnix int64, c Constraints) (Result, error)
}
