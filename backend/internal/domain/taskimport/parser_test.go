package taskimport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormat(t *testing.T) {
	cases := map[string]Format{
		"tasks.csv": FormatCSV,
		"TASKS.CSV": FormatCSV,
		"data.xlsx": FormatXLSX,
		"data.XLS":  FormatXLSX,
		"data.xls":  FormatXLSX,
		"readme.md": FormatUnknown,
		"":          FormatUnknown,
		"noext":     FormatUnknown,
		"  t.csv  ": FormatCSV,
	}
	for name, want := range cases {
		assert.Equal(t, want, DetectFormat(name), name)
	}
}

func TestParse_UnsupportedFormat(t *testing.T) {
	_, err := Parse([]byte("x"), FormatUnknown)
	assert.ErrorIs(t, err, UnsupportedFormatError)
}

func TestParse_CSV_HappyPath(t *testing.T) {
	csvData := []byte("title,address,duration,date,window_start,window_end\n" +
		"Встреча,\"Москва, ул. Арбат, 1\",45,25.12.2025,10:00,12:00\n" +
		"Доставка,\"Москва, Тверская, 10\",20,2025-12-26,14:00,\n" +
		"Отчёт,,60,,,\n")

	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 3)
	assert.Empty(t, res.Errors)

	// Row 1 — полный набор полей
	r1 := res.Rows[0]
	assert.Equal(t, "Встреча", r1.Title)
	assert.Equal(t, "Москва, ул. Арбат, 1", r1.AddressText)
	require.NotNil(t, r1.DurationMin)
	assert.Equal(t, 45, *r1.DurationMin)
	require.NotNil(t, r1.Window.StartDate)
	assert.Equal(t, "2025-12-25", *r1.Window.StartDate)
	require.NotNil(t, r1.Window.StartTime)
	assert.Equal(t, "10:00", *r1.Window.StartTime)
	require.NotNil(t, r1.Window.EndTime)
	assert.Equal(t, "12:00", *r1.Window.EndTime)

	// Row 2 — ISO-дата, только начало окна
	r2 := res.Rows[1]
	assert.Equal(t, "Доставка", r2.Title)
	require.NotNil(t, r2.Window.StartDate)
	assert.Equal(t, "2025-12-26", *r2.Window.StartDate)
	assert.Nil(t, r2.Window.EndTime)
	assert.Nil(t, r2.Window.EndDate)

	// Row 3 — без адреса, без окна; дата — сегодня (не сверяем точно)
	r3 := res.Rows[2]
	assert.Equal(t, "Отчёт", r3.Title)
	assert.Equal(t, "", r3.AddressText)
	require.NotNil(t, r3.DurationMin)
	assert.Equal(t, 60, *r3.DurationMin)
	assert.Nil(t, r3.Window.StartTime)
	assert.Nil(t, r3.Window.StartDate)
}

func TestParse_CSV_WithBOM(t *testing.T) {
	csvData := append([]byte{0xEF, 0xBB, 0xBF},
		[]byte("title,duration\nЗадача,15\n")...)
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "Задача", res.Rows[0].Title)
}

func TestParse_CSV_ValidationErrors(t *testing.T) {
	csvData := []byte("title,duration,date,window_start,window_end\n" +
		",10,,,\n" + // пустой title
		"A,abc,,,\n" + // duration не число
		"B,0,,,\n" + // duration < 1
		"C,,32.13.2025,,\n" + // невалидная дата
		"D,,,abc,,\n" + // невалидное время
		"E,,,10:00,09:00\n") // end < start

	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	assert.Empty(t, res.Rows)
	require.Len(t, res.Errors, 6)

	// Row numbers — 1-based среди непустых строк данных
	for i, rowErr := range res.Errors {
		assert.Equal(t, i+1, rowErr.Row)
		assert.NotEmpty(t, rowErr.Errors)
	}

	assert.Contains(t, res.Errors[0].Errors[0], "title")
	assert.Contains(t, res.Errors[1].Errors[0], "duration")
	assert.Contains(t, res.Errors[2].Errors[0], "duration")
	assert.Contains(t, res.Errors[3].Errors[0], "date")
	// row 5: неправильное время + возможно проверка порядка окна, допускаем ≥1 ошибку
	assert.Contains(t, res.Errors[4].Errors[0], "window_start")
	assert.Contains(t, res.Errors[5].Errors[0], "позже")
}

func TestParse_CSV_EmptyRows(t *testing.T) {
	csvData := []byte("title,duration\n" +
		",\n" + // полностью пустая — пропускаем
		"A,10\n" +
		",,\n") // пустая — пропускаем
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "A", res.Rows[0].Title)
}

func TestParse_CSV_HeadersCaseAndSpaces(t *testing.T) {
	// Заголовки с разным регистром и пробелами должны нормализоваться.
	csvData := []byte("  TITLE , Window Start\nA,09:00\n")
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "A", res.Rows[0].Title)
	require.NotNil(t, res.Rows[0].Window.StartTime)
	assert.Equal(t, "09:00", *res.Rows[0].Window.StartTime)
}

func TestParse_CSV_DefaultDateIsToday(t *testing.T) {
	csvData := []byte("title,window_start\nA,09:00\n")
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	require.NotNil(t, res.Rows[0].Window.StartDate)
	assert.Equal(t, time.Now().Format("2006-01-02"), *res.Rows[0].Window.StartDate)
}

func TestParse_CSV_TimeNormalization(t *testing.T) {
	csvData := []byte("title,window_start,window_end\nA,9:05,17:3\n")
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	// "17:3" не проходит regex (нужны 2 цифры после ':') → ошибка
	require.NotEmpty(t, res.Errors)
	// "9:05" приемлемо (regex допускает 1-2 цифры часа)
	// но поскольку window_end невалиден, строка полностью в errors
}

func TestParse_CSV_TimeNormalization_Single(t *testing.T) {
	csvData := []byte("title,window_start\nA,9:05\n")
	res, err := Parse(csvData, FormatCSV)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	require.NotNil(t, res.Rows[0].Window.StartTime)
	assert.Equal(t, "09:05", *res.Rows[0].Window.StartTime)
}

func TestParse_CSV_Empty(t *testing.T) {
	res, err := Parse([]byte(""), FormatCSV)
	require.NoError(t, err)
	assert.Empty(t, res.Rows)
	assert.Empty(t, res.Errors)
}
