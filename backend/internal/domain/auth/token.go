package auth

import "time"

type AccessClaims struct {
	UserID string
	Email  string
	Exp    time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
