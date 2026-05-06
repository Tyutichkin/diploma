package http

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"planner-backend/internal/domain/task"
	importstory "planner-backend/internal/domain/taskimport/story"
)

const maxImportFileSize = 10 << 20 // 10 MB

type ImportHandlers struct {
	story *importstory.Story
}

func NewImportHandlers(st *importstory.Story) *ImportHandlers {
	return &ImportHandlers{story: st}
}

func (h *ImportHandlers) Import(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImportFileSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file (possibly too large)"})
		return
	}

	startSortIndex := 0
	if v := c.Query("startSortIndex"); v != "" {
		if n, convErr := strconv.Atoi(v); convErr == nil {
			startSortIndex = n
		}
	}

	result, err := h.story.Import(c.Request.Context(), userID, data, header.Filename, startSortIndex)
	if err != nil {
		var valErr *task.ValidationError
		if errors.As(err, &valErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": valErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	resp := ImportTasksResp{
		Imported: result.Imported,
		Errors:   make([]ImportRowErrorDTO, len(result.Errors)),
	}
	for i, e := range result.Errors {
		resp.Errors[i] = ImportRowErrorDTO{Row: e.Row, Title: e.Title, Errors: e.Errors}
	}

	c.JSON(http.StatusOK, resp)
}
