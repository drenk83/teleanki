package postgres

import (
	"context"
	"errors"
	"time"

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
RETURNING ` + userCols

	u, err := scanUser(r.pool.QueryRow(ctx, q, telegramID, username))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE telegram_id = $1`, telegramID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) SetDailyLimit(ctx context.Context, userID int64, limit int) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `
UPDATE users SET daily_limit = $2, updated_at = now()
WHERE id = $1
RETURNING `+userCols, userID, limit))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) AddReview(ctx context.Context, userID int64, today time.Time) (*domain.User, error) {
	day := domain.DayDate(today)
	u, err := scanUser(r.pool.QueryRow(ctx, `
UPDATE users SET
    reviews_on_date = $2,
    reviews_today = CASE WHEN reviews_on_date = $2 THEN reviews_today + 1 ELSE 1 END,
    updated_at = now()
WHERE id = $1
RETURNING `+userCols, userID, day))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) LearnDeckIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT deck_id FROM user_learn_decks WHERE user_id = $1 ORDER BY deck_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *UserRepository) ReplaceLearnDecks(ctx context.Context, userID int64, deckIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM user_learn_decks WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(deckIDs) > 0 {
		if _, err := tx.Exec(ctx, `
INSERT INTO user_learn_decks (user_id, deck_id)
SELECT $1, id FROM decks
WHERE id = ANY($2)
  AND (user_id = $1 OR id IN (SELECT deck_id FROM deck_members WHERE user_id = $1))`, userID, deckIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *UserRepository) SetLearnFree(ctx context.Context, userID int64, free bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET learn_free = $2, updated_at = now() WHERE id = $1`, userID, free)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *UserRepository) SetNotify(ctx context.Context, userID int64, enabled bool, hour int) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `
UPDATE users SET notify_enabled = $2, notify_hour = $3, updated_at = now()
WHERE id = $1
RETURNING `+userCols, userID, enabled, hour))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) MarkNotified(ctx context.Context, userID int64, day time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET notify_on_date = $2, updated_at = now() WHERE id = $1`, userID, domain.DayDate(day))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *UserRepository) ListForNotify(ctx context.Context, hour int, day time.Time) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
SELECT `+userCols+`
FROM users
WHERE notify_enabled = true
  AND notify_hour = $1
  AND notify_on_date <> $2`, hour, domain.DayDate(day))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
