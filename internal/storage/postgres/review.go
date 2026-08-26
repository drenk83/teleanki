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

type ReviewRepository struct {
	pool *pgxpool.Pool
}

func NewReviewRepository(db *DB) *ReviewRepository {
	return &ReviewRepository{pool: db.Pool}
}

func (r *ReviewRepository) GetByCardID(ctx context.Context, cardID int64) (*domain.Review, error) {
	const q = `
SELECT card_id, easiness, interval_days, repetitions, due_at, updated_at
FROM reviews
WHERE card_id = $1`

	var rev domain.Review
	err := r.pool.QueryRow(ctx, q, cardID).Scan(
		&rev.CardID,
		&rev.Easiness,
		&rev.IntervalDays,
		&rev.Repetitions,
		&rev.DueAt,
		&rev.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *ReviewRepository) Apply(ctx context.Context, review *domain.Review, userID int64, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const upd = `
UPDATE reviews r
SET easiness = $3, interval_days = $4, repetitions = $5, due_at = $6, updated_at = now()
FROM cards c
JOIN decks d ON d.id = c.deck_id
WHERE r.card_id = $1
  AND c.id = r.card_id
  AND d.user_id = $2
RETURNING r.updated_at`
	err = tx.QueryRow(ctx, upd,
		review.CardID,
		userID,
		review.Easiness,
		review.IntervalDays,
		review.Repetitions,
		review.DueAt,
	).Scan(&review.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	if err != nil {
		return err
	}

	day := utcDate(now)
	tag, err := tx.Exec(ctx, `
UPDATE users SET
    reviews_on_date = $2,
    reviews_today = CASE WHEN reviews_on_date = $2 THEN reviews_today + 1 ELSE 1 END,
    updated_at = now()
WHERE id = $1`, userID, day)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *ReviewRepository) Update(ctx context.Context, review *domain.Review) error {
	const q = `
UPDATE reviews
SET easiness = $2, interval_days = $3, repetitions = $4, due_at = $5, updated_at = now()
WHERE card_id = $1
RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q,
		review.CardID,
		review.Easiness,
		review.IntervalDays,
		review.Repetitions,
		review.DueAt,
	).Scan(&review.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	return err
}

const dueItemCols = `
    c.id, c.deck_id, c.front, c.back, c.mode, c.choices, c.reversible, c.created_at, c.updated_at,
    d.id, d.user_id, d.name, d.created_at, d.updated_at,
    r.card_id, r.easiness, r.interval_days, r.repetitions, r.due_at, r.updated_at`

func (r *ReviewRepository) ListDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time, limit int) ([]storage.DueItem, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	q := `
SELECT` + dueItemCols + `
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE d.user_id = $1
  AND r.due_at <= $2
  AND (cardinality($3::bigint[]) = 0 OR d.id = ANY($3::bigint[]))
ORDER BY r.due_at ASC, c.id ASC`
	var rows pgx.Rows
	var err error
	if limit > 0 {
		q += `
LIMIT $4`
		rows, err = r.pool.Query(ctx, q, userID, now, deckIDs, limit)
	} else {
		rows, err = r.pool.Query(ctx, q, userID, now, deckIDs)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectDueItems(rows)
}

func (r *ReviewRepository) ListForLearn(ctx context.Context, userID int64, deckIDs []int64) ([]storage.DueItem, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	q := `
SELECT` + dueItemCols + `
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE d.user_id = $1
  AND (cardinality($2::bigint[]) = 0 OR d.id = ANY($2::bigint[]))
ORDER BY c.id ASC`

	rows, err := r.pool.Query(ctx, q, userID, deckIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectDueItems(rows)
}

func collectDueItems(rows pgx.Rows) ([]storage.DueItem, error) {
	var out []storage.DueItem
	for rows.Next() {
		item, err := scanDueItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
