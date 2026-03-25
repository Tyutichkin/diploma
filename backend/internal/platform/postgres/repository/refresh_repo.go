package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/auth"
)

type RefreshRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshRepo(pool *pgxpool.Pool) *RefreshRepo {
	return &RefreshRepo{pool: pool}
}

func (r *RefreshRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (auth.RefreshTokenRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO refresh_tokens(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at
	`, userID, tokenHash, expiresAt)

	var rec auth.RefreshTokenRecord
	if err := row.Scan(&rec.ID, &rec.UserID, &rec.TokenHash, &rec.ExpiresAt, &rec.RevokedAt, &rec.CreatedAt); err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	return rec, nil
}

func (r *RefreshRepo) GetActiveByHash(ctx context.Context, tokenHash string) (auth.RefreshTokenRecord, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash=$1
	`, tokenHash)

	var rec auth.RefreshTokenRecord
	if err := row.Scan(&rec.ID, &rec.UserID, &rec.TokenHash, &rec.ExpiresAt, &rec.RevokedAt, &rec.CreatedAt); err != nil {
		return auth.RefreshTokenRecord{}, false, nil
	}

	if rec.RevokedAt != nil || time.Now().After(rec.ExpiresAt) {
		return auth.RefreshTokenRecord{}, false, nil
	}
	return rec, true, nil
}

func (r *RefreshRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at=$1
		WHERE token_hash=$2 AND revoked_at IS NULL
	`, now, tokenHash)
	return err
}
