package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authstory "planner-backend/internal/domain/auth/story"
)

const ctxUserIDKey = "userID"
const ctxEmailKey = "email"

func AuthMiddleware(auth *authstory.Story) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token := strings.TrimPrefix(h, "Bearer ")
		claims, err := auth.VerifyAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxEmailKey, claims.Email)
		c.Next()
	}
}

func userIDFromCtx(c *gin.Context) (string, bool) {
	v, exists := c.Get(ctxUserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	return s, true
}
