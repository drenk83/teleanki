package postgres

import (
	"context"
	"errors"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{pool: db.Pool}
}

func (r *UserRepository) UpsertByTelegramID(ctx context.Context, telegramID int64, username string) (*domain.User, error) {
	const q = `
INSERT INTO users (telegram_id, username)
VALUES ($1, $2)
ON CONFLICT (telegram_id) DO UPDATE SET
    username = EXCLUDED.username,
    updated_at = now()
RETURNING id, telegram_id, username, created_at, updated_at`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, telegramID, username).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	const q = `
SELECT id, telegram_id, username, created_at, updated_at
FROM users
WHERE telegram_id = $1`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
