package task

import "time"

// Task — задача пользователя с геопривязкой.
type Task struct {
	ID          string
	UserID      string
	Title       string
	AddressText string
	Latitude    float64
	Longitude   float64

	DurationMin int
	WindowStart *string // "HH:MM:SS" (проще для интеграции с фронтом)
	WindowEnd   *string

	SortIndex int

	CreatedAt time.Time
	UpdatedAt time.Time
	IsDeleted bool
}
