package task

import "time"

type Task struct {
	ID              string
	UserID          string
	Title           string
	AddressText     string
	Latitude        *float64
	Longitude       *float64
	DurationMin     *int
	WindowStartDate *string
	WindowStartTime *string
	WindowEndDate   *string
	WindowEndTime   *string
	SortIndex       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	IsCompleted     bool
}
