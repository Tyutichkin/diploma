package taskimport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// readXLSX разбирает .xlsx / .xls (OOXML) минимальным ридером на stdlib.
// Первый лист используется как источник данных; первая строка — заголовки.
func readXLSX(data []byte) ([]map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("xlsx: not a valid archive: %w", err)
	}

	var (
		shared     []string
		sheetBytes []byte
		sheetName  string
	)

	for _, f := range zr.File {
		switch {
		case f.Name == "xl/sharedStrings.xml":
			b, err := readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("xlsx: read sharedStrings: %w", err)
			}
			shared, err = parseSharedStrings(b)
			if err != nil {
				return nil, fmt.Errorf("xlsx: parse sharedStrings: %w", err)
			}

		case strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml"):
			// Берём лексикографически первый файл листа (обычно sheet1.xml).
			if sheetName == "" || f.Name < sheetName {
				b, err := readZipFile(f)
				if err != nil {
					return nil, fmt.Errorf("xlsx: read sheet: %w", err)
				}
				sheetName = f.Name
				sheetBytes = b
			}
		}
	}

	if sheetBytes == nil {
		return nil, fmt.Errorf("xlsx: no worksheet found")
	}

	grid, err := parseSheet(sheetBytes, shared)
	if err != nil {
		return nil, fmt.Errorf("xlsx: parse sheet: %w", err)
	}
	if len(grid) == 0 {
		return nil, nil
	}

	headers := grid[0]
	var result []map[string]string
	for _, row := range grid[1:] {
		if isAllEmpty(row) {
			continue
		}
		m := make(map[string]string, len(headers))
		for i, h := range headers {
			key := strings.TrimSpace(h)
			if key == "" {
				continue
			}
			val := ""
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}
			m[key] = val
		}
		result = append(result, m)
	}
	return result, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

type xlsxSST struct {
	XMLName xml.Name `xml:"sst"`
	Items   []xlsxSI `xml:"si"`
}

// xlsxSI — элемент sharedStrings. Поддерживаем обычный <t> и rich-text <r><t>.
type xlsxSI struct {
	T string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (s xlsxSI) text() string {
	if s.T != "" || len(s.R) == 0 {
		return s.T
	}
	var b strings.Builder
	for _, r := range s.R {
		b.WriteString(r.T)
	}
	return b.String()
}

type xlsxWorksheet struct {
	XMLName   xml.Name `xml:"worksheet"`
	SheetData struct {
		Rows []xlsxRow `xml:"row"`
	} `xml:"sheetData"`
}

type xlsxRow struct {
	R     int        `xml:"r,attr"`
	Cells []xlsxCell `xml:"c"`
}

// xlsxCell — одна ячейка. T — тип: "s" (sharedString), "str" (формульная строка),
// "inlineStr" (встроенная строка), "b" (bool), иначе число.
type xlsxCell struct {
	R  string `xml:"r,attr"`
	T  string `xml:"t,attr"`
	V  string `xml:"v"`
	Is struct {
		T string `xml:"t"`
	} `xml:"is"`
}

func parseSharedStrings(b []byte) ([]string, error) {
	var sst xlsxSST
	if err := xml.Unmarshal(b, &sst); err != nil {
		return nil, err
	}
	out := make([]string, len(sst.Items))
	for i, si := range sst.Items {
		out[i] = si.text()
	}
	return out, nil
}

func parseSheet(b []byte, shared []string) ([][]string, error) {
	var ws xlsxWorksheet
	if err := xml.Unmarshal(b, &ws); err != nil {
		return nil, err
	}
	var grid [][]string
	for _, row := range ws.SheetData.Rows {
		cells := map[int]string{}
		maxCol := -1
		for _, c := range row.Cells {
			col := colFromRef(c.R)
			if col < 0 {
				continue
			}
			if col > maxCol {
				maxCol = col
			}
			cells[col] = cellValue(c, shared)
		}
		rowArr := make([]string, maxCol+1)
		for i := range rowArr {
			rowArr[i] = cells[i]
		}
		grid = append(grid, rowArr)
	}
	return grid, nil
}

func cellValue(c xlsxCell, shared []string) string {
	switch c.T {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(c.V))
		if err != nil || idx < 0 || idx >= len(shared) {
			return ""
		}
		return shared[idx]
	case "inlineStr":
		return c.Is.T
	default:
		// "str", "b", "n", "" — все трактуем как сырое значение V.
		return c.V
	}
}

// colFromRef: "A1" → 0, "B2" → 1, "AA3" → 26, "" → -1.
func colFromRef(ref string) int {
	col := 0
	for i := 0; i < len(ref); i++ {
		r := ref[i]
		switch {
		case r >= 'A' && r <= 'Z':
			col = col*26 + int(r-'A'+1)
		case r >= 'a' && r <= 'z':
			col = col*26 + int(r-'a'+1)
		default:
			if col == 0 {
				return -1
			}
			return col - 1
		}
	}
	if col == 0 {
		return -1
	}
	return col - 1
}
