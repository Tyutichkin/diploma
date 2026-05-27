package task

import "time"

// TimeWindow — желаемый интервал обслуживания задачи.
// Каждое поле может быть nil; в Update nil-поле означает «не изменять»,
// "" (пустая строка) — «очистить».
type TimeWindow struct {
	StartDate *string // "YYYY-MM-DD"
	StartTime *string // "HH:MM"
	EndDate   *string // "YYYY-MM-DD"
	EndTime   *string // "HH:MM"
}

func (w TimeWindow) IsEmpty() bool {
	return w.StartDate == nil && w.StartTime == nil && w.EndDate == nil && w.EndTime == nil
}

type Task struct {
	ID          string
	UserID      string
	Title       string
	AddressText string
	Latitude    *float64
	Longitude   *float64
	DurationMin *int
	Window      TimeWindow
	SortIndex   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsCompleted bool
}
