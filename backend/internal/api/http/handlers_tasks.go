package http

import (
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
		respondError(c, err)
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
		respondBadRequest(c, "bad request")
		return
	}

	out, err := h.story.Create(c.Request.Context(), userID, req.toCreateInput())
	if err != nil {
		respondError(c, err)
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
		respondBadRequest(c, "bad request")
		return
	}

	if len(req.Tasks) == 0 {
		respondBadRequest(c, "tasks must not be empty")
		return
	}

	inputs := make([]task.CreateInput, len(req.Tasks))
	for i, t := range req.Tasks {
		inputs[i] = t.toCreateInput()
	}

	out, err := h.story.CreateBatch(c.Request.Context(), userID, inputs)
	if err != nil {
		respondError(c, err)
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
		respondBadRequest(c, "bad request")
		return
	}

	out, found, err := h.story.Update(c.Request.Context(), userID, taskID, req.toUpdateInput())
	if err != nil {
		respondError(c, err)
		return
	}
	if !found {
		respondNotFound(c)
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
		respondBadRequest(c, "bad request")
		return
	}

	items := make([]task.ReorderItem, len(req.Order))
	for i, o := range req.Order {
		items[i] = task.ReorderItem{TaskID: o.ID, SortIndex: o.SortIndex}
	}

	if err := h.story.Reorder(c.Request.Context(), userID, task.ReorderInput{Items: items}); err != nil {
		respondError(c, err)
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
		respondError(c, err)
		return
	}
	if !found {
		respondNotFound(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *TaskHandlers) DeleteAll(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	deleted, err := h.story.DeleteAll(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": deleted})
}
