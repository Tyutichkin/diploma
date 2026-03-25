package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"planner-backend/internal/domain/user"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash string) (user.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users(email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`, email, passwordHash)

	var u user.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return user.User{}, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (user.User, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email=$1
	`, email)

	var u user.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return user.User{}, false, nil
	}
	return u, true, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (user.User, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id=$1
	`, id)

	var u user.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return user.User{}, false, nil
	}
	return u, true, nil
}
