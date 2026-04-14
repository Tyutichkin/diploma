package ptrs

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T { return &v }
