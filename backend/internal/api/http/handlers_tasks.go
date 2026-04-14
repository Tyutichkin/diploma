package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"planner-backend/internal/domain/task"
	taskstory "planner-backend/internal/domain/task/story"
)

type TaskHandlers struct {
	story *taskstory.Story
}

func NewTaskHandlers(st *taskstory.Story) *TaskHandlers {
	return &TaskHandlers{story: st}
}

func (h *TaskHandlers) List(c *gin.Context) {
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

func (h *TaskHandlers) Create(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	var addressText string
	if req.AddressText != nil {
		addressText = *req.AddressText
	}

	out, err := h.story.Create(c.Request.Context(), userID, task.CreateInput{
		Title:           req.Title,
		AddressText:     addressText,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		DurationMin:     req.DurationMin,
		WindowStartDate: req.WindowStartDate,
		WindowStartTime: req.WindowStartTime,
		WindowEndDate:   req.WindowEndDate,
		WindowEndTime:   req.WindowEndTime,
		SortIndex:       req.SortIndex,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *TaskHandlers) BatchCreate(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req BatchCreateTasksReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	if len(req.Tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tasks must not be empty"})
		return
	}

	inputs := make([]task.CreateInput, len(req.Tasks))
	for i, t := range req.Tasks {
		var addressText string
		if t.AddressText != nil {
			addressText = *t.AddressText
		}
		inputs[i] = task.CreateInput{
			Title:           t.Title,
			AddressText:     addressText,
			Latitude:        t.Latitude,
			Longitude:       t.Longitude,
			DurationMin:     t.DurationMin,
			WindowStartDate: t.WindowStartDate,
			WindowStartTime: t.WindowStartTime,
			WindowEndDate:   t.WindowEndDate,
			WindowEndTime:   t.WindowEndTime,
			SortIndex:       t.SortIndex,
		}
	}

	out, err := h.story.CreateBatch(c.Request.Context(), userID, inputs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *TaskHandlers) Update(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	taskID := c.Param("id")

	var req UpdateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	out, found, err := h.story.Update(c.Request.Context(), userID, taskID, task.UpdateInput{
		Title:           req.Title,
		AddressText:     req.AddressText,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		DurationMin:     req.DurationMin,
		WindowStartDate: req.WindowStartDate,
		WindowStartTime: req.WindowStartTime,
		WindowEndDate:   req.WindowEndDate,
		WindowEndTime:   req.WindowEndTime,
		SortIndex:       req.SortIndex,
		IsCompleted:     req.IsCompleted,
	})
	if err != nil {
		var valErr *task.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": valErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *TaskHandlers) Reorder(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req ReorderTasksReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	items := make([]task.ReorderItem, len(req.Order))
	for i, o := range req.Order {
		items[i] = task.ReorderItem{TaskID: o.ID, SortIndex: o.SortIndex}
	}

	if err := h.story.Reorder(c.Request.Context(), userID, task.ReorderInput{Items: items}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *TaskHandlers) Delete(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	taskID := c.Param("id")

	found, err := h.story.Delete(c.Request.Context(), userID, taskID)
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
