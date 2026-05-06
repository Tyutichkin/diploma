package task

import "time"

// Task — задача пользователя. Поля Latitude/Longitude могут быть nil,
// если задача создана без адреса (нет геопривязки).
type Task struct {
	ID          string
	UserID      string
	Title       string
	AddressText string   // пустая строка = нет адреса
	Latitude    *float64 // nil = нет координат
	Longitude   *float64 // nil = нет координат

	DurationMin *int // nil = задача не занимает времени (мгновенная)

	// Раздельные дата и время для временных окон.
	// Дата: "YYYY-MM-DD"; Время: "HH:MM" или "HH:MM:SS".
	// nil = не задано.
	WindowStartDate *string
	WindowStartTime *string
	WindowEndDate   *string
	WindowEndTime   *string

	SortIndex int

	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsCompleted bool
}
