package story

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	dauth "planner-backend/internal/domain/auth"
	"planner-backend/internal/domain/user"
)

type mockUserRepo struct {
	createFn     func(ctx context.Context, email, passwordHash string) (user.User, error)
	getByEmailFn func(ctx context.Context, email string) (user.User, bool, error)
	getByIDFn    func(ctx context.Context, id string) (user.User, bool, error)
}

func (m *mockUserRepo) Create(ctx context.Context, email, passwordHash string) (user.User, error) {
	return m.createFn(ctx, email, passwordHash)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (user.User, bool, error) {
	return m.getByEmailFn(ctx, email)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string) (user.User, bool, error) {
	return m.getByIDFn(ctx, id)
}

type mockRefreshRepo struct {
	createFn          func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (dauth.RefreshTokenRecord, error)
	getActiveByHashFn func(ctx context.Context, tokenHash string) (dauth.RefreshTokenRecord, bool, error)
	revokeByHashFn    func(ctx context.Context, tokenHash string) error
}

func (m *mockRefreshRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (dauth.RefreshTokenRecord, error) {
	return m.createFn(ctx, userID, tokenHash, expiresAt)
}
func (m *mockRefreshRepo) GetActiveByHash(ctx context.Context, tokenHash string) (dauth.RefreshTokenRecord, bool, error) {
	return m.getActiveByHashFn(ctx, tokenHash)
}
func (m *mockRefreshRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	return m.revokeByHashFn(ctx, tokenHash)
}

const testSecret = "super-test-secret-key"

func newTestStory(u *mockUserRepo, r *mockRefreshRepo) *Story {
	return New(u, r, testSecret, 15, 30)
}

func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func makeBcryptHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func testUser(id, email, passwordHash string) user.User {
	return user.User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func okRefreshRecord(userID, tokenHash string) dauth.RefreshTokenRecord {
	return dauth.RefreshTokenRecord{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
}

// 1.1.1 Успешная регистрация
func TestRegister_Success(t *testing.T) {
	uid := uuid.NewString()
	email := "user@example.com"
	password := "pass123"

	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			return testUser(uid, e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	s := newTestStory(uRepo, rRepo)
	pair, err := s.Register(context.Background(), email, password)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

// 1.1.2 Дублирующий email
func TestRegister_DuplicateEmail(t *testing.T) {
	uid := uuid.NewString()
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return testUser(uid, "user@example.com", "hash"), true, nil
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Register(context.Background(), "user@example.com", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// 1.1.6 Пароль хэшируется bcrypt
func TestRegister_PasswordHashedWithBcrypt(t *testing.T) {
	password := "secret123"
	var capturedHash string

	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			capturedHash = h
			return testUser(uuid.NewString(), e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	s := newTestStory(uRepo, rRepo)
	_, err := s.Register(context.Background(), "a@b.com", password)
	require.NoError(t, err)

	assert.NotEqual(t, password, capturedHash, "пароль не должен лежать в открытом виде")
	err = bcrypt.CompareHashAndPassword([]byte(capturedHash), []byte(password))
	assert.NoError(t, err, "сохранённый хеш должен быть bcrypt от пароля")
}

// 1.2.1 Успешный вход
func TestLogin_Success(t *testing.T) {
	password := "correctpass"
	uid := uuid.NewString()
	hash := makeBcryptHash(t, password)

	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return testUser(uid, "user@example.com", hash), true, nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	s := newTestStory(uRepo, rRepo)
	pair, err := s.Login(context.Background(), "user@example.com", password)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

// 1.2.2 Пользователь не найден
func TestLogin_UserNotFound(t *testing.T) {
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Login(context.Background(), "nobody@example.com", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// 1.2.3 Неверный пароль
func TestLogin_WrongPassword(t *testing.T) {
	hash := makeBcryptHash(t, "correctpass")
	uid := uuid.NewString()

	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return testUser(uid, "user@example.com", hash), true, nil
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Login(context.Background(), "user@example.com", "wrongpass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// 1.2.4 Пустой email
func TestLogin_EmptyEmail(t *testing.T) {
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Login(context.Background(), "", "pass")
	assert.Error(t, err)
}

// 1.2.6 SQL-инъекция в email
func TestLogin_SQLInjectionEmail(t *testing.T) {
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Login(context.Background(), "' OR '1'='1", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// 1.3.1 Успешное обновление токена
func TestRefresh_Success(t *testing.T) {
	uid := uuid.NewString()
	rawToken := "some-refresh-token"
	th := hashOf(rawToken)

	uRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id string) (user.User, bool, error) {
			return testUser(id, "user@example.com", "hash"), true, nil
		},
	}
	rRepo := &mockRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return okRefreshRecord(uid, th), true, nil
		},
		revokeByHashFn: func(_ context.Context, _ string) error { return nil },
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	s := newTestStory(uRepo, rRepo)
	pair, err := s.Refresh(context.Background(), rawToken)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

// 1.3.2 Истёкший refresh-токен
func TestRefresh_ExpiredToken(t *testing.T) {
	rRepo := &mockRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return dauth.RefreshTokenRecord{}, false, nil
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)
	_, err := s.Refresh(context.Background(), "expired-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// 1.3.3 Отозванный refresh-токен
func TestRefresh_RevokedToken(t *testing.T) {
	rRepo := &mockRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return dauth.RefreshTokenRecord{}, false, nil // отозван или просрочен
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)
	_, err := s.Refresh(context.Background(), "revoked-token")
	assert.Error(t, err)
}

// 1.3.4 Несуществующий refresh-токен
func TestRefresh_NonExistentToken(t *testing.T) {
	rRepo := &mockRefreshRepo{
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			return dauth.RefreshTokenRecord{}, false, nil
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)
	_, err := s.Refresh(context.Background(), "random-nonexistent-string")
	assert.Error(t, err)
}

// 1.3.6 Повторное использование отозванного токена (после Logout)
func TestRefresh_AfterLogout(t *testing.T) {
	revokeCount := 0
	rRepo := &mockRefreshRepo{
		revokeByHashFn: func(_ context.Context, _ string) error {
			revokeCount++
			return nil
		},
		getActiveByHashFn: func(_ context.Context, _ string) (dauth.RefreshTokenRecord, bool, error) {
			if revokeCount > 0 {
				return dauth.RefreshTokenRecord{}, false, nil
			}
			return okRefreshRecord("uid", "hash"), true, nil
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)

	_ = s.Logout(context.Background(), "some-token")
	_, err := s.Refresh(context.Background(), "some-token")
	assert.Error(t, err)
}

// 1.4.1 Успешный выход
func TestLogout_Success(t *testing.T) {
	revokedHash := ""
	rRepo := &mockRefreshRepo{
		revokeByHashFn: func(_ context.Context, th string) error {
			revokedHash = th
			return nil
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)
	token := "my-refresh-token"
	err := s.Logout(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, hashOf(token), revokedHash)
}

// 1.4.2 Двойной выход
func TestLogout_Idempotent(t *testing.T) {
	rRepo := &mockRefreshRepo{
		revokeByHashFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	s := newTestStory(&mockUserRepo{}, rRepo)
	token := "some-token"
	err1 := s.Logout(context.Background(), token)
	err2 := s.Logout(context.Background(), token)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

// 1.5.1 Валидный токен
func TestVerifyAccessToken_Valid(t *testing.T) {
	uid := uuid.NewString()
	email := "user@example.com"

	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			return testUser(uid, e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	s := newTestStory(uRepo, rRepo)
	pair, err := s.Register(context.Background(), email, "pass123")
	require.NoError(t, err)

	claims, err := s.VerifyAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uid, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.True(t, claims.Exp.After(time.Now()))
}

// 1.5.2 истёкший токен.
func TestVerifyAccessToken_Expired(t *testing.T) {
	// TTL = -1 мин → токен протух мгновенно.
	s := New(&mockUserRepo{}, &mockRefreshRepo{}, testSecret, -1, 30)
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			return testUser(uuid.NewString(), e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}
	s2 := New(uRepo, rRepo, testSecret, -1, 30)
	pair, err := s2.Register(context.Background(), "a@b.com", "pass")
	require.NoError(t, err)

	_, err = s.VerifyAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

// 1.5.3 Поддельная подпись
func TestVerifyAccessToken_TamperedSignature(t *testing.T) {
	s := newTestStory(&mockUserRepo{}, &mockRefreshRepo{})
	// JWT, подписанный другим секретом.
	otherStory := New(&mockUserRepo{}, &mockRefreshRepo{}, "other-secret", 15, 30)
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			return testUser(uuid.NewString(), e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}
	otherStory.users = uRepo
	otherStory.refresh = rRepo

	pair, err := otherStory.Register(context.Background(), "a@b.com", "pass")
	require.NoError(t, err)

	_, err = s.VerifyAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

// 1.5.4 Случайная строка
func TestVerifyAccessToken_RandomString(t *testing.T) {
	s := newTestStory(&mockUserRepo{}, &mockRefreshRepo{})
	_, err := s.VerifyAccessToken("randomstring")
	assert.Error(t, err)
}

// 1.5.5 Пустая строка
func TestVerifyAccessToken_EmptyString(t *testing.T) {
	s := newTestStory(&mockUserRepo{}, &mockRefreshRepo{})
	_, err := s.VerifyAccessToken("")
	assert.Error(t, err)
}

// 1.5.6 Токен с другим секретом
func TestVerifyAccessToken_DifferentSecret(t *testing.T) {
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, nil
		},
		createFn: func(_ context.Context, e, h string) (user.User, error) {
			return testUser(uuid.NewString(), e, h), nil
		},
	}
	rRepo := &mockRefreshRepo{
		createFn: func(_ context.Context, _, _ string, _ time.Time) (dauth.RefreshTokenRecord, error) {
			return dauth.RefreshTokenRecord{}, nil
		},
	}

	storyOtherSecret := New(uRepo, rRepo, "other-secret", 15, 30)
	pair, err := storyOtherSecret.Register(context.Background(), "a@b.com", "pass")
	require.NoError(t, err)

	storyMySecret := newTestStory(&mockUserRepo{}, &mockRefreshRepo{})
	_, err = storyMySecret.VerifyAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestRegister_RepoError(t *testing.T) {
	uRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (user.User, bool, error) {
			return user.User{}, false, errors.New("db error")
		},
	}
	s := newTestStory(uRepo, &mockRefreshRepo{})
	_, err := s.Register(context.Background(), "a@b.com", "pass")
	assert.Error(t, err)
}
