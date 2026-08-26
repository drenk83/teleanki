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

func (r *ReviewRepository) ListDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time, limit int) ([]storage.DueItem, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	const q = `
SELECT
    c.id, c.deck_id, c.front, c.back, c.mode, c.choices, c.created_at, c.updated_at,
    d.id, d.user_id, d.name, d.created_at, d.updated_at,
    r.card_id, r.easiness, r.interval_days, r.repetitions, r.due_at, r.updated_at
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE d.user_id = $1
  AND r.due_at <= $2
  AND (cardinality($3::bigint[]) = 0 OR d.id = ANY($3::bigint[]))
ORDER BY r.due_at ASC, c.id ASC
LIMIT $4`

	rows, err := r.pool.Query(ctx, q, userID, now, deckIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []storage.DueItem
	for rows.Next() {
		var item storage.DueItem
		var cardMode string
		err := rows.Scan(
			&item.Card.ID,
			&item.Card.DeckID,
			&item.Card.Front,
			&item.Card.Back,
			&cardMode,
			&item.Card.Choices,
			&item.Card.CreatedAt,
			&item.Card.UpdatedAt,
			&item.Deck.ID,
			&item.Deck.UserID,
			&item.Deck.Name,
			&item.Deck.CreatedAt,
			&item.Deck.UpdatedAt,
			&item.Review.CardID,
			&item.Review.Easiness,
			&item.Review.IntervalDays,
			&item.Review.Repetitions,
			&item.Review.DueAt,
			&item.Review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		item.Card.Mode = domain.Mode(cardMode)
		if item.Card.Choices == nil {
			item.Card.Choices = []string{}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
