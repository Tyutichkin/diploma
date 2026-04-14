package taskimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UnsupportedFormatError возвращается, если формат файла не распознан.
var UnsupportedFormatError = errors.New("unsupported file format: use .csv, .xlsx or .xls")

// Parse разбирает содержимое файла согласно формату и валидирует строки.
// Ошибки отдельных строк не приводят к ошибке функции — они попадают в ParseResult.Errors.
func Parse(data []byte, format Format) (ParseResult, error) {
	var (
		rows []map[string]string
		err  error
	)
	switch format {
	case FormatCSV:
		rows, err = readCSV(data)
	case FormatXLSX:
		rows, err = readXLSX(data)
	default:
		return ParseResult{}, UnsupportedFormatError
	}
	if err != nil {
		return ParseResult{}, err
	}
	return validateRows(rows), nil
}

func readCSV(data []byte) ([]map[string]string, error) {
	data = stripBOM(data)
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // разрешаем разное число колонок на строку

	records, err := r.ReadAll()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	var result []map[string]string
	for _, rec := range records[1:] {
		if isAllEmpty(rec) {
			continue
		}
		m := make(map[string]string, len(headers))
		for i, h := range headers {
			key := strings.TrimSpace(h)
			if key == "" {
				continue
			}
			if i >= len(rec) {
				m[key] = ""
				continue
			}
			m[key] = strings.TrimSpace(rec[i])
		}
		result = append(result, m)
	}
	return result, nil
}

func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

func isAllEmpty(rec []string) bool {
	for _, v := range rec {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

var timeRe = regexp.MustCompile(`^\d{1,2}:\d{2}$`)

func validateRows(rows []map[string]string) ParseResult {
	res := ParseResult{}
	today := time.Now().Format("2006-01-02")

	for i, raw := range rows {
		rowNum := i + 1
		m := normalizeKeys(raw)
		var errs []string

		title := m["title"]
		if title == "" {
			errs = append(errs, "title — обязательное поле")
		}

		parsed := ParsedRow{Title: title}

		if addr := m["address"]; addr != "" {
			parsed.AddressText = addr
		}

		if dur := m["duration"]; dur != "" {
			d, err := strconv.Atoi(dur)
			if err != nil || d < 1 {
				errs = append(errs, fmt.Sprintf("duration должно быть целым числом ≥ 1 (получено: %q)", dur))
			} else {
				parsed.DurationMin = &d
			}
		}

		// Единое поле date применяется к обеим границам окна.
		dateRaw := m["date"]
		var windowDate string
		switch {
		case dateRaw == "":
			windowDate = today
		default:
			iso, ok := parseDateValue(dateRaw)
			if !ok {
				errs = append(errs, fmt.Sprintf(
					"date должна быть в формате ДД.ММ.ГГГГ или ГГГГ-ММ-ДД (получено: %q)", dateRaw))
				windowDate = today
			} else {
				windowDate = iso
			}
		}

		ws := m["window_start"]
		we := m["window_end"]
		if ws != "" && !timeRe.MatchString(ws) {
			errs = append(errs, fmt.Sprintf("window_start должно быть в формате ЧЧ:ММ (получено: %q)", ws))
			ws = ""
		}
		if we != "" && !timeRe.MatchString(we) {
			errs = append(errs, fmt.Sprintf("window_end должно быть в формате ЧЧ:ММ (получено: %q)", we))
			we = ""
		}
		if ws != "" && we != "" && normalizeTime(ws) >= normalizeTime(we) {
			errs = append(errs, "window_end должен быть позже window_start")
		}

		if ws != "" {
			t := normalizeTime(ws)
			parsed.WindowStartTime = &t
			d := windowDate
			parsed.WindowStartDate = &d
		}
		if we != "" {
			t := normalizeTime(we)
			parsed.WindowEndTime = &t
			d := windowDate
			parsed.WindowEndDate = &d
		}

		if len(errs) > 0 {
			res.Errors = append(res.Errors, RowError{
				Row:    rowNum,
				Title:  title,
				Errors: errs,
			})
			continue
		}
		res.Rows = append(res.Rows, parsed)
	}
	return res
}

// normalizeKeys приводит ключи к нижнему регистру и заменяет пробелы на "_".
func normalizeKeys(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[normalizeHeader(k)] = strings.TrimSpace(v)
	}
	return out
}

func normalizeHeader(h string) string {
	s := strings.ToLower(strings.TrimSpace(h))
	return strings.Join(strings.Fields(s), "_")
}

// parseDateValue принимает ДД.ММ.ГГГГ или ГГГГ-ММ-ДД.
// Возвращает ISO-формат ГГГГ-ММ-ДД.
func parseDateValue(value string) (string, bool) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t.Format("2006-01-02"), true
	}
	if t, err := time.Parse("02.01.2006", value); err == nil {
		return t.Format("2006-01-02"), true
	}
	return "", false
}

// normalizeTime приводит "9:05" → "09:05" (так же делает фронтенд при отправке).
func normalizeTime(t string) string {
	parts := strings.SplitN(t, ":", 2)
	if len(parts) != 2 {
		return t
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return t
	}
	return fmt.Sprintf("%02d:%02d", h, m)
}
