package distance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"planner-backend/internal/common/timing"
)

// OSRMProvider — клиент к OSRM Table API
// (http://project-osrm.org/docs/v5.24.0/api/#table-service).
type OSRMProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewOSRMProvider — если baseURL пустой, использует http://router.project-osrm.org.
func NewOSRMProvider(baseURL string) *OSRMProvider {
	if baseURL == "" {
		baseURL = "http://router.project-osrm.org"
	}
	return &OSRMProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type osrmTableResp struct {
	Code      string      `json:"code"`
	Durations [][]float64 `json:"durations"` // секунды; null — недостижимо
	Distances [][]float64 `json:"distances"` // метры; null — недостижимо
}

// GetMatrix — вызов OSRM Table и построение n×n матрицы. Недостижимым парам
// ставится большое значение, чтобы оптимизатор мог построить маршрут.
func (p *OSRMProvider) GetMatrix(ctx context.Context, points []Point) ([][]Edge, error) {
	n := len(points)
	if n == 0 {
		return nil, nil
	}

	// OSRM ждёт координаты в порядке "lng,lat".
	coords := make([]string, n)
	for i, pt := range points {
		coords[i] = fmt.Sprintf("%.6f,%.6f", pt.Lng, pt.Lat)
	}

	url := fmt.Sprintf(
		"%s/table/v1/driving/%s?annotations=duration,distance",
		p.baseURL, strings.Join(coords, ";"),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("osrm: build request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osrm: http: %w", err)
	}
	defer resp.Body.Close()

	var body osrmTableResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("osrm: decode response: %w", err)
	}

	if body.Code != "Ok" {
		return nil, fmt.Errorf("osrm: api error: %s", body.Code)
	}

	matrix := make([][]Edge, n)
	for i := range matrix {
		matrix[i] = make([]Edge, n)
		for j := range matrix[i] {
			matrix[i][j] = edgeFromOSRM(body.Durations, body.Distances, i, j)
		}
	}
	return matrix, nil
}

// edgeFromOSRM — NaN и отрицательные значения (недостижимые пары) заменяются большим fallback.
func edgeFromOSRM(durations, distances [][]float64, i, j int) Edge {
	const unreachableDurSec = 99_999 // ~27 часов — заведомо избегается оптимизатором

	durSec := unreachableDurSec
	if durations != nil && i < len(durations) && j < len(durations[i]) {
		if v := durations[i][j]; !math.IsNaN(v) && v >= 0 {
			durSec = int(math.Round(v))
		}
	}

	distM := 0
	if distances != nil && i < len(distances) && j < len(distances[i]) {
		if v := distances[i][j]; !math.IsNaN(v) && v >= 0 {
			distM = int(math.Round(v))
		}
	}

	// Округляем длительность вверх до целой минуты, чтобы дальше система
	// работала только с минутно-выровненными секундами (см. internal/common/timing).
	return Edge{DistanceM: distM, DurationSec: timing.RoundUpSecToMinute(durSec)}
}
