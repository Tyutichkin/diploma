package story

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	dauth "planner-backend/internal/domain/auth"
	authgate "planner-backend/internal/domain/auth/gate"
)

type Story struct {
	users   authgate.UserRepository
	refresh authgate.RefreshTokenRepository

	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New(users authgate.UserRepository, refresh authgate.RefreshTokenRepository, jwtSecret string, accessTTLMin int, refreshTTLDays int) *Story {
	return &Story{
		users:      users,
		refresh:    refresh,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  time.Duration(accessTTLMin) * time.Minute,
		refreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (s *Story) Register(ctx context.Context, email, password string) (dauth.TokenPair, error) {
	if _, ok, err := s.users.GetByEmail(ctx, email); err != nil {
		return dauth.TokenPair{}, err
	} else if ok {
		return dauth.TokenPair{}, errors.New("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dauth.TokenPair{}, err
	}

	u, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		return dauth.TokenPair{}, err
	}

	return s.issueTokens(ctx, u.ID, u.Email)
}

func (s *Story) Login(ctx context.Context, email, password string) (dauth.TokenPair, error) {
	u, ok, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return dauth.TokenPair{}, err
	}
	if !ok {
		return dauth.TokenPair{}, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return dauth.TokenPair{}, errors.New("invalid credentials")
	}

	return s.issueTokens(ctx, u.ID, u.Email)
}

func (s *Story) Refresh(ctx context.Context, refreshToken string) (dauth.TokenPair, error) {
	th := hashToken(refreshToken)

	rec, ok, err := s.refresh.GetActiveByHash(ctx, th)
	if err != nil {
		return dauth.TokenPair{}, err
	}
	if !ok {
		return dauth.TokenPair{}, errors.New("refresh token invalid")
	}

	u, ok, err := s.users.GetByID(ctx, rec.UserID)
	if err != nil {
		return dauth.TokenPair{}, err
	}
	if !ok {
		return dauth.TokenPair{}, errors.New("user not found")
	}

	_ = s.refresh.RevokeByHash(ctx, th)

	return s.issueTokens(ctx, u.ID, u.Email)
}

func (s *Story) Logout(ctx context.Context, refreshToken string) error {
	return s.refresh.RevokeByHash(ctx, hashToken(refreshToken))
}

func (s *Story) VerifyAccessToken(tokenStr string) (dauth.AccessClaims, error) {
	t, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !t.Valid {
		return dauth.AccessClaims{}, errors.New("invalid token")
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return dauth.AccessClaims{}, errors.New("invalid claims")
	}

	uid, _ := claims["uid"].(string)
	email, _ := claims["email"].(string)
	expFloat, _ := claims["exp"].(float64)
	exp := time.Unix(int64(expFloat), 0)

	if uid == "" || email == "" {
		return dauth.AccessClaims{}, errors.New("invalid claims")
	}

	return dauth.AccessClaims{UserID: uid, Email: email, Exp: exp}, nil
}

func (s *Story) issueTokens(ctx context.Context, userID, email string) (dauth.TokenPair, error) {
	access, err := s.makeJWT(userID, email)
	if err != nil {
		return dauth.TokenPair{}, err
	}

	rawRefresh := uuid.NewString() + "-" + uuid.NewString()
	expires := time.Now().Add(s.refreshTTL)

	if _, err := s.refresh.Create(ctx, userID, hashToken(rawRefresh), expires); err != nil {
		return dauth.TokenPair{}, err
	}

	return dauth.TokenPair{AccessToken: access, RefreshToken: rawRefresh}, nil
}

func (s *Story) makeJWT(userID, email string) (string, error) {
	now := time.Now()
	exp := now.Add(s.accessTTL)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
	})

	return token.SignedString(s.jwtSecret)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
