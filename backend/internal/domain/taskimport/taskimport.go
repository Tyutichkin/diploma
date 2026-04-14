// Package taskimport реализует разбор файлов CSV/XLSX с задачами
// для массового импорта. Валидация правил соответствует клиентскому
// TaskImport.tsx: те же обязательные поля, те же форматы дат и времени.
package taskimport

import "strings"

// ParsedRow — валидированная строка импорта, готовая к созданию task.CreateInput.
// Все поля, кроме Title, опциональны.
type ParsedRow struct {
	Title           string
	AddressText     string  // "" если адрес не указан
	DurationMin     *int    // nil если длительность не задана
	WindowStartDate *string // "YYYY-MM-DD"
	WindowStartTime *string // "HH:MM"
	WindowEndDate   *string
	WindowEndTime   *string
}

// RowError описывает ошибку конкретной строки исходного файла.
// Row — 1-based индекс среди непустых строк данных (без учёта заголовка).
type RowError struct {
	Row    int
	Title  string
	Errors []string
}

// ParseResult — разбитый на валидные и невалидные строки результат разбора.
type ParseResult struct {
	Rows   []ParsedRow
	Errors []RowError
}

// Format — поддерживаемый формат файла импорта.
type Format int

const (
	FormatUnknown Format = iota
	FormatCSV
	FormatXLSX
)

// DetectFormat определяет формат по имени файла (расширению).
func DetectFormat(filename string) Format {
	lower := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(lower, ".csv"):
		return FormatCSV
	case strings.HasSuffix(lower, ".xlsx"), strings.HasSuffix(lower, ".xls"):
		return FormatXLSX
	default:
		return FormatUnknown
	}
}
