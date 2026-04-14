package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	dauth "planner-backend/internal/domain/auth"
	authstory "planner-backend/internal/domain/auth/story"
	"planner-backend/internal/domain/user"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testUserRepo struct {
	createFn     func(ctx context.Context, email, passwordHash string) (user.User, error)
	getByEmailFn func(ctx context.Context, email string) (user.User, bool, error)
	getByIDFn    func(ctx context.Context, id string) (user.User, bool, error)
}

func (m *testUserRepo) Create(ctx context.Context, email, passwordHash string) (user.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, email, passwordHash)
	}
	return user.User{}, nil
}
func (m *testUserRepo) GetByEmail(ctx context.Context, email string) (user.User, bool, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return user.User{}, false, nil
}
func (m *testUserRepo) GetByID(ctx context.Context, id string) (user.User, bool, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return user.User{}, false, nil
}

type testRefreshRepo struct {
	createFn          func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (dauth.RefreshTokenRecord, error)
	getActiveByHashFn func(ctx context.Context, tokenHash string) (dauth.RefreshTokenRecord, bool, error)
	revokeByHashFn    func(ctx context.Context, tokenHash string) error
}

func (m *testRefreshRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (dauth.RefreshTokenRecord, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, tokenHash, expiresAt)
	}
	return dauth.RefreshTokenRecord{}, nil
}
func (m *testRefreshRepo) GetActiveByHash(ctx context.Context, tokenHash string) (dauth.RefreshTokenRecord, bool, error) {
	if m.getActiveByHashFn != nil {
		return m.getActiveByHashFn(ctx, tokenHash)
	}
	return dauth.RefreshTokenRecord{}, false, nil
}
func (m *testRefreshRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	if m.revokeByHashFn != nil {
		return m.revokeByHashFn(ctx, tokenHash)
	}
	return nil
}

const handlerTestSecret = "handler-test-secret"

func newTestAuthStory(u *testUserRepo, r *testRefreshRepo) *authstory.Story {
	return authstory.New(u, r, handlerTestSecret, 15, 30)
}

func newAuthRouter(auth *authstory.Story) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	authH := NewAuthHandlers(auth)
	api := r.Group("/api")
	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)
	api.POST("/auth/logout", authH.Logout)
	return r
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func makeBcryptHashH(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// 5.1.1 POST /api/auth/register — успех
func TestHandler_Register_Success(t *testing.T) {
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
	router := newAuthRouter(newTestAuthStory(uRepo, rRepo))

	w := doJSON(t, router, "POST", "/api/auth/register", map[string]string{
		"email": "a@b.com", "password": "secret123",
	})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp TokenPairResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

// 5.1.2 POST /api/auth/register — дубликат → 409
func TestHandler_Register_Duplicate(t *testing.T) {
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{ID: "uid", Email: "a@b.com"}, true, nil
		},
	}
	router := newAuthRouter(newTestAuthStory(uRepo, &testRefreshRepo{}))
	w := doJSON(t, router, "POST", "/api/auth/register", map[string]string{
		"email": "a@b.com", "password": "secret",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

// 5.1.3 POST /api/auth/register — пустое тело → 400
func TestHandler_Register_EmptyBody(t *testing.T) {
	router := newAuthRouter(newTestAuthStory(&testUserRepo{}, &testRefreshRepo{}))
	w := doJSON(t, router, "POST", "/api/auth/register", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5.1.4 POST /api/auth/register — некорректный JSON → 400
func TestHandler_Register_BrokenJSON(t *testing.T) {
	router := newAuthRouter(newTestAuthStory(&testUserRepo{}, &testRefreshRepo{}))
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString("{broken"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5.1.5 POST /api/auth/login — успех
func TestHandler_Login_Success(t *testing.T) {
	password := "testpassword"
	hash := makeBcryptHashH(t, password)
	uid := uuid.NewString()

	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, email string) (user.User, bool, error) {
			return user.User{ID: uid, Email: email, PasswordHash: hash}, true, nil
		},
	}
	rRepo := &testRefreshRepo{}
	router := newAuthRouter(newTestAuthStory(uRepo, rRepo))

	w := doJSON(t, router, "POST", "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": password,
	})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp TokenPairResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.AccessToken)
}

// 5.1.6 POST /api/auth/login — неверный пароль → 401
func TestHandler_Login_WrongPassword(t *testing.T) {
	hash := makeBcryptHashH(t, "correctpassword")
	uid := uuid.NewString()

	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{ID: uid, Email: "a@b.com", PasswordHash: hash}, true, nil
		},
	}
	router := newAuthRouter(newTestAuthStory(uRepo, &testRefreshRepo{}))
	w := doJSON(t, router, "POST", "/api/auth/login", map[string]string{
		"email": "a@b.com", "password": "wrongpassword",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.1.7 POST /api/auth/login — пользователь не найден → 401
func TestHandler_Login_UserNotFound(t *testing.T) {
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
	}
	router := newAuthRouter(newTestAuthStory(uRepo, &testRefreshRepo{}))
	w := doJSON(t, router, "POST", "/api/auth/login", map[string]string{
		"email": "nobody@example.com", "password": "pass",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.1.8 POST /api/auth/refresh — успех
func TestHandler_Refresh_Success(t *testing.T) {
	uid := uuid.NewString()
	rawToken := "valid-refresh-token-" + uuid.NewString()

	uRepo := &testUserRepo{
		getByIDFn: func(_ context.Context, id string) (user.User, bool, error) {
			return user.User{ID: id, Email: "user@example.com"}, true, nil
		},
	}
	rRepo := &testRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return dauth.RefreshTokenRecord{
				ID:        uuid.NewString(),
				UserID:    uid,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, true, nil
		},
		revokeByHashFn: func(_ context.Context, _ string) error { return nil },
	}
	router := newAuthRouter(newTestAuthStory(uRepo, rRepo))

	w := doJSON(t, router, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": rawToken,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp TokenPairResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

// 5.1.9 POST /api/auth/refresh — истёкший токен → 401
func TestHandler_Refresh_ExpiredToken(t *testing.T) {
	rRepo := &testRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return dauth.RefreshTokenRecord{}, false, nil
		},
	}
	router := newAuthRouter(newTestAuthStory(&testUserRepo{}, rRepo))
	w := doJSON(t, router, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": "expired-token",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 5.1.10 POST /api/auth/logout — успех → 200 {"ok":true}
func TestHandler_Logout_Success(t *testing.T) {
	rRepo := &testRefreshRepo{
		revokeByHashFn: func(_ context.Context, _ string) error { return nil },
	}
	router := newAuthRouter(newTestAuthStory(&testUserRepo{}, rRepo))
	w := doJSON(t, router, "POST", "/api/auth/logout", map[string]string{
		"refreshToken": "some-token",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, true, resp["ok"])
}

func TestHandler_Register_RepoCreateError(t *testing.T) {
	uRepo := &testUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, _, _ string) (user.User, error) {
			return user.User{}, errors.New("db error")
		},
	}
	router := newAuthRouter(newTestAuthStory(uRepo, &testRefreshRepo{}))
	w := doJSON(t, router, "POST", "/api/auth/register", map[string]string{
		"email": "a@b.com", "password": "pass",
	})
	assert.NotEqual(t, http.StatusOK, w.Code)
}
