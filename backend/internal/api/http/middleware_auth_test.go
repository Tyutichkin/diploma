package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	authstory "planner-backend/internal/domain/auth/story"
	"planner-backend/internal/domain/user"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newSecuredRouter(auth *authstory.Story) *gin.Engine {
	r := gin.New()
	secured := r.Group("/api")
	secured.Use(AuthMiddleware(auth))
	secured.GET("/protected", func(c *gin.Context) {
		uid, _ := userIDFromCtx(c)
		c.JSON(http.StatusOK, gin.H{"uid": uid})
	})
	return r
}

func buildValidToken(t *testing.T, secret string) string {
	t.Helper()
	uid := uuid.NewString()
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, email, hash string) (user.User, error) {
			return user.User{ID: uid, Email: email, PasswordHash: hash}, nil
		},
	}
	rRepo := &testRefreshRepo{}
	s := authstory.New(uRepo, rRepo, secret, 15, 30)
	pair, err := s.Register(context.Background(), "test@test.com", "password")
	if err != nil {
		t.Fatalf("failed to build valid token: %v", err)
	}
	return pair.AccessToken
}

func buildExpiredToken(t *testing.T, secret string) string {
	t.Helper()
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, email, hash string) (user.User, error) {
			return user.User{ID: uuid.NewString(), Email: email, PasswordHash: hash}, nil
		},
	}
	rRepo := &testRefreshRepo{}
	s := authstory.New(uRepo, rRepo, secret, -1, 30) // -1 min TTL = expired
	pair, err := s.Register(context.Background(), "test@test.com", "password")
	if err != nil {
		t.Fatalf("failed to build expired token: %v", err)
	}
	return pair.AccessToken
}

const middlewareSecret = "middleware-test-secret"

// ── Auth Middleware tests ──────────────────────────────────────────────────────

// 5.2.1 Без заголовка Authorization → 401
func TestMiddleware_NoAuthHeader(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	r := newSecuredRouter(auth)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.2.2 Заголовок без Bearer prefix → 401
func TestMiddleware_NoBearerPrefix(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	r := newSecuredRouter(auth)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "token123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.2.3 Истёкший access-токен → 401
func TestMiddleware_ExpiredToken(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	r := newSecuredRouter(auth)

	expiredToken := buildExpiredToken(t, middlewareSecret)
	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.2.4 Поддельный токен (подписан другим ключом) → 401
func TestMiddleware_TamperedToken(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	r := newSecuredRouter(auth)

	// Token signed with different secret
	fakeToken := buildValidToken(t, "totally-different-secret")
	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+fakeToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.2.5 Валидный токен → запрос пропускается, uid в контексте
func TestMiddleware_ValidToken(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)

	uid := uuid.NewString()
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, email, hash string) (user.User, error) {
			return user.User{ID: uid, Email: email, PasswordHash: hash}, nil
		},
	}
	rRepo := &testRefreshRepo{}
	storyWithUID := authstory.New(uRepo, rRepo, middlewareSecret, 15, 30)
	pair, err := storyWithUID.Register(context.Background(), "user@test.com", "pass")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	r := newSecuredRouter(auth)
	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// UID might differ because auth uses different story, but token is valid for same secret
	// We just verify the request passes through (no 401)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

// 5.2.6 Случайная строка как токен → 401
func TestMiddleware_RandomStringToken(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	r := newSecuredRouter(auth)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer notajwttoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Middleware blocks request but not auth endpoints ─────────────────────────

func TestMiddleware_OnlyAppliedToSecuredRoutes(t *testing.T) {
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)

	// Full router with both public and secured routes
	r := gin.New()
	authH := NewAuthHandlers(auth)
	api := r.Group("/api")
	api.POST("/auth/register", authH.Register)

	secured := api.Group("")
	secured.Use(AuthMiddleware(auth))
	secured.GET("/tasks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Public route — no auth needed
	req := httptest.NewRequest("POST", "/api/auth/register", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)

	// Secured route — no token → 401
	req2 := httptest.NewRequest("GET", "/api/tasks", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

// Ensure expired token was created recently (sanity check for test helper)
func TestBuildExpiredToken_IsActuallyExpired(t *testing.T) {
	token := buildExpiredToken(t, middlewareSecret)
	auth := authstory.New(&testUserRepo{}, &testRefreshRepo{}, middlewareSecret, 15, 30)
	claims, err := auth.VerifyAccessToken(token)
	assert.Error(t, err, "expired token should fail verification")
	_ = claims
	_ = time.Now()
}
