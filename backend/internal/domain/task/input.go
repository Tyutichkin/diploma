package task

type CreateInput struct {
	Title       string
	AddressText string
	Latitude    float64
	Longitude   float64
	DurationMin int
	WindowStart *string
	WindowEnd   *string
	SortIndex   int
}

type UpdateInput struct {
	Title       *string
	AddressText *string
	Latitude    *float64
	Longitude   *float64
	DurationMin *int
	WindowStart *string
	WindowEnd   *string
	SortIndex   *int
}

type ReorderItem struct {
	TaskID    string
	SortIndex int
}

type ReorderInput struct {
	Items []ReorderItem
}
