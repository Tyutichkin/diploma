package task

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type CreateInput struct {
	Title       string
	AddressText string   // пустая строка = нет адреса
	Latitude    *float64 // nil = нет координат
	Longitude   *float64 // nil = нет координат
	DurationMin *int     // nil = не задана (мгновенная)
	Window      TimeWindow
	SortIndex   int
}

// UpdateInput: nil-поля не изменяются. Для Window — nil означает «не трогать окно».
// Когда Window != nil, внутри каждого *string поля действует та же семантика:
// nil — не трогать; "" — очистить; иначе — задать.
type UpdateInput struct {
	Title       *string
	AddressText *string
	Latitude    *float64
	Longitude   *float64
	DurationMin *int
	Window      *TimeWindow
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
