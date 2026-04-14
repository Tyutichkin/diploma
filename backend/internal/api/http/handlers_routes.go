package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	routestory "planner-backend/internal/domain/route/story"
	"planner-backend/internal/platform/distance"
)

type RouteHandlers struct {
	story *routestory.Story
}

func NewRouteHandlers(st *routestory.Story) *RouteHandlers {
	return &RouteHandlers{story: st}
}

func (h *RouteHandlers) Create(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req CreateRouteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	out, err := h.story.CreateDraft(c.Request.Context(), userID, req.Source, req.OrderedTaskIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *RouteHandlers) List(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	out, err := h.story.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Optimize — POST /api/routes/optimize. Ответ такой же, как у GET /api/routes/:id.
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

	var matrix [][]distance.Edge
	if len(req.DistanceMatrix) > 0 {
		matrix = make([][]distance.Edge, len(req.DistanceMatrix))
		for i, row := range req.DistanceMatrix {
			matrix[i] = make([]distance.Edge, len(row))
			for j, cell := range row {
				matrix[i][j] = distance.Edge{
					DistanceM:   cell.DistanceM,
					DurationSec: cell.DurationSec,
				}
			}
		}
	}

	precedences := make([]routestory.PrecedenceConstraint, len(req.PrecedenceConstraints))
	for i, p := range req.PrecedenceConstraints {
		precedences[i] = routestory.PrecedenceConstraint{
			BeforeTaskID: p.BeforeTaskID,
			AfterTaskID:  p.AfterTaskID,
		}
	}

	out, err := h.story.Optimize(c.Request.Context(), userID, req.TaskIDs, startTime, matrix, req.StartTaskID, req.EndTaskID, precedences)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *RouteHandlers) Get(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	routeID := c.Param("id")

	out, found, err := h.story.Get(c.Request.Context(), userID, routeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *RouteHandlers) Delete(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	routeID := c.Param("id")

	found, err := h.story.Delete(c.Request.Context(), userID, routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RouteHandlers) DeleteAll(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	if err := h.story.DeleteAll(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RouteHandlers) Rename(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	routeID := c.Param("id")

	var req RenameRouteReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	found, err := h.story.Rename(c.Request.Context(), userID, routeID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
