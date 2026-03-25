package auth

import "time"

// AccessClaims — данные, которые кладём в JWT.
type AccessClaims struct {
	UserID string
	Email  string
	Exp    time.Time
}

// TokenPair — пара токенов для фронта.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
