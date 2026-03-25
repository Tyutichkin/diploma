package gate

import (
	"context"
	"time"

	"planner-backend/internal/domain/auth"
	"planner-backend/internal/domain/user"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (user.User, error)
	GetByEmail(ctx context.Context, email string) (user.User, bool, error)
	GetByID(ctx context.Context, id string) (user.User, bool, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (auth.RefreshTokenRecord, error)
	GetActiveByHash(ctx context.Context, tokenHash string) (auth.RefreshTokenRecord, bool, error)
	RevokeByHash(ctx context.Context, tokenHash string) error
}
