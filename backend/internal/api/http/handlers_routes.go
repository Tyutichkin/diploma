package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"planner-backend/internal/common/timing"
	routestory "planner-backend/internal/domain/route/story"
	"planner-backend/internal/platform/distance"
)

type RouteHandlers struct {
	story *routestory.Story
}

func NewRouteHandlers(st *routestory.Story) *RouteHandlers {
	return &RouteHandlers{story: st}
}

// Optimize — POST /api/routes/optimize. Перезаписывает единственный сохранённый
// маршрут пользователя (см. RouteRepo.SaveOptimizedRoute) и возвращает его целиком.
func (h *RouteHandlers) Optimize(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req OptimizeRouteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	startTime := req.StartTimeUnix
	if startTime <= 0 {
		startTime = time.Now().Unix()
	}
	// Округляем стартовое время вверх до целой минуты — система оперирует только
	// минутно-выровненными секундами.
	startTime = timing.RoundUpUnixToMinute(startTime)

	matrix := convertMatrix(req.DistanceMatrix)
	precedences := convertPrecedences(req.PrecedenceConstraints)

	out, err := h.story.Optimize(c.Request.Context(), userID, req.TaskIDs, startTime, matrix, req.StartTaskID, req.EndTaskID, precedences)
	if err != nil {
		// API-контракт: все ошибки оптимизации поднимаются как 400 — фронт показывает их
		// текст пользователю. Если в будущем потребуется отделять внутренние сбои,
		// нужно вернуть типизированную ошибку из routestory.Optimize.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// convertMatrix конвертирует DTO-матрицу клиента в формат distance.Edge.
// Возвращает nil, если матрица пуста (бэкенд тогда сам построит её через провайдера).
func convertMatrix(in [][]DistanceCellDTO) [][]distance.Edge {
	if len(in) == 0 {
		return nil
	}
	out := make([][]distance.Edge, len(in))
	for i, row := range in {
		out[i] = make([]distance.Edge, len(row))
		for j, cell := range row {
			out[i][j] = distance.Edge{
				DistanceM: cell.DistanceM,
				// Защита от клиентов, которые могут прислать «грязные» секунды:
				// внутри проекта длительности всегда минутно-выровнены.
				DurationSec: timing.RoundUpSecToMinute(cell.DurationSec),
			}
		}
	}
	return out
}

func convertPrecedences(in []PrecedencePairDTO) []routestory.PrecedenceConstraint {
	out := make([]routestory.PrecedenceConstraint, len(in))
	for i, p := range in {
		out[i] = routestory.PrecedenceConstraint{
			BeforeTaskID: p.BeforeTaskID,
			AfterTaskID:  p.AfterTaskID,
		}
	}
	return out
}
