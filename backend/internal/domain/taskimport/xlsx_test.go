package taskimport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildXLSX создаёт минимальный OOXML-файл в памяти.
// Первая строка grid — заголовки. Все значения идут как sharedStrings,
// что ближе всего к тому, как пишут Excel и xlsx-библиотеки (например, `xlsx` на фронте).
func buildXLSX(t *testing.T, grid [][]string) []byte {
	t.Helper()

	// Собираем уникальные строки и мэппинг string→index.
	var shared []string
	idx := map[string]int{}
	for _, row := range grid {
		for _, cell := range row {
			if _, ok := idx[cell]; !ok {
				idx[cell] = len(shared)
				shared = append(shared, cell)
			}
		}
	}

	// sharedStrings.xml
	var sst strings.Builder
	fmt.Fprintf(&sst, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
		`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`,
		len(shared), len(shared))
	for _, s := range shared {
		fmt.Fprintf(&sst, `<si><t xml:space="preserve">%s</t></si>`, escapeXML(s))
	}
	sst.WriteString(`</sst>`)

	// sheet1.xml
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	sheet.WriteString(`<sheetData>`)
	for i, row := range grid {
		rowNum := i + 1
		fmt.Fprintf(&sheet, `<row r="%d">`, rowNum)
		for j, cell := range row {
			ref := colRef(j) + fmt.Sprintf("%d", rowNum)
			fmt.Fprintf(&sheet, `<c r="%s" t="s"><v>%d</v></c>`, ref, idx[cell])
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	// Workbook / content types / rels — минимум, достаточный для нашего ридера,
	// которому хватает sharedStrings.xml и xl/worksheets/sheet*.xml.
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	add := func(name, content string) {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	add("xl/sharedStrings.xml", sst.String())
	add("xl/worksheets/sheet1.xml", sheet.String())

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// colRef возвращает буквенный столбец для 0-based индекса: 0→A, 25→Z, 26→AA.
func colRef(i int) string {
	var b []byte
	i++
	for i > 0 {
		i--
		b = append([]byte{byte('A' + i%26)}, b...)
		i /= 26
	}
	return string(b)
}

func TestColFromRef(t *testing.T) {
	cases := map[string]int{
		"A1":  0,
		"B2":  1,
		"Z10": 25,
		"AA1": 26,
		"AB1": 27,
		"":    -1,
		"1":   -1,
	}
	for ref, want := range cases {
		assert.Equal(t, want, colFromRef(ref), ref)
	}
}

func TestParse_XLSX_HappyPath(t *testing.T) {
	xlsxBytes := buildXLSX(t, [][]string{
		{"title", "address", "duration", "date", "window_start", "window_end"},
		{"Встреча", "Москва, ул. Арбат, 1", "45", "25.12.2025", "10:00", "12:00"},
		{"Отчёт", "", "60", "", "", ""},
	})

	res, err := Parse(xlsxBytes, FormatXLSX)
	require.NoError(t, err)
	require.Len(t, res.Rows, 2)
	assert.Empty(t, res.Errors)

	assert.Equal(t, "Встреча", res.Rows[0].Title)
	require.NotNil(t, res.Rows[0].DurationMin)
	assert.Equal(t, 45, *res.Rows[0].DurationMin)
	require.NotNil(t, res.Rows[0].WindowStartDate)
	assert.Equal(t, "2025-12-25", *res.Rows[0].WindowStartDate)

	assert.Equal(t, "Отчёт", res.Rows[1].Title)
	assert.Equal(t, "", res.Rows[1].AddressText)
}

func TestParse_XLSX_InvalidArchive(t *testing.T) {
	_, err := Parse([]byte("not a zip"), FormatXLSX)
	assert.Error(t, err)
}

func TestParse_XLSX_SkipsEmptyRows(t *testing.T) {
	xlsxBytes := buildXLSX(t, [][]string{
		{"title", "duration"},
		{"", ""},
		{"A", "10"},
	})
	res, err := Parse(xlsxBytes, FormatXLSX)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "A", res.Rows[0].Title)
}
