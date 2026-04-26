// Package ptrs содержит универсальные вспомогательные функции для работы с указателями.
package ptrs

// Ptr возвращает указатель на v.
func Ptr[T any](v T) *T { return &v }

// Deref возвращает значение из p или нулевое значение типа, если p == nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// DerefOr возвращает значение из p или fallback, если p == nil.
func DerefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
