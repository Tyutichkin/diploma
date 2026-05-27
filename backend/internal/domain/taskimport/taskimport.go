package taskimport

import "strings"

type ParsedRow struct {
	Title           string
	AddressText     string
	DurationMin     *int
	WindowStartDate *string
	WindowStartTime *string
	WindowEndDate   *string
	WindowEndTime   *string
}

type RowError struct {
	Row    int
	Title  string
	Errors []string
}

type ParseResult struct {
	Rows   []ParsedRow
	Errors []RowError
}

type Format int

const (
	FormatUnknown Format = iota
	FormatCSV
	FormatXLSX
)

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
