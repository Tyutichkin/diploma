package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	authstory "planner-backend/internal/domain/auth/story"
)

type AuthHandlers struct {
	auth *authstory.Story
}

func NewAuthHandlers(auth *authstory.Story) *AuthHandlers {
	return &AuthHandlers{auth: auth}
}

func (h *AuthHandlers) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	pair, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TokenPairResp{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	pair, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, TokenPairResp{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandlers) Refresh(c *gin.Context) {
	var req RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	pair, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed"})
		return
	}

	c.JSON(http.StatusOK, TokenPairResp{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}

func (h *AuthHandlers) Logout(c *gin.Context) {
	var req RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	_ = h.auth.Logout(c.Request.Context(), req.RefreshToken)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
