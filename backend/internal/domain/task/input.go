package task

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type CreateInput struct {
	Title       string
	AddressText string   // пустая строка = нет адреса
	Latitude    *float64 // nil = нет координат
	Longitude   *float64 // nil = нет координат
	DurationMin *int     // nil = не задана (мгновенная)

	WindowStartDate *string // "YYYY-MM-DD"
	WindowStartTime *string // "HH:MM"
	WindowEndDate   *string // "YYYY-MM-DD"
	WindowEndTime   *string // "HH:MM"

	SortIndex int
}

type UpdateInput struct {
	Title       *string
	AddressText *string
	Latitude    *float64
	Longitude   *float64
	DurationMin *int

	WindowStartDate *string
	WindowStartTime *string
	WindowEndDate   *string
	WindowEndTime   *string

	SortIndex   *int
	IsCompleted *bool
}

type ReorderItem struct {
	TaskID    string
	SortIndex int
}

type ReorderInput struct {
	Items []ReorderItem
}
