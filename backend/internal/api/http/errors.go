package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"planner-backend/internal/domain/task"
)

// respondError мапит domain-ошибки в HTTP-ответ. По умолчанию — 500;
// task.ValidationError превращается в 400 с сообщением для клиента.
func respondError(c *gin.Context, err error) {
	var valErr *task.ValidationError
	if errors.As(err, &valErr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": valErr.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
}

func respondBadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func respondNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
