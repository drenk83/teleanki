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

func (r *ReviewRepository) Apply(ctx context.Context, review *domain.Review, userID int64, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrap("apply review", err)
	}
	defer tx.Rollback(ctx)

	const upd = `
UPDATE reviews r
SET easiness = $3, interval_days = $4, repetitions = $5, due_at = $6, updated_at = now()
FROM cards c
JOIN decks d ON d.id = c.deck_id
WHERE r.card_id = $1
  AND r.user_id = $2
  AND c.id = r.card_id
  AND (d.user_id = $2 OR EXISTS (
      SELECT 1 FROM deck_members m WHERE m.deck_id = d.id AND m.user_id = $2
  ))
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
		return wrap("apply review", err)
	}

	day := domain.DayDate(now)
	tag, err := tx.Exec(ctx, `
UPDATE users SET
    reviews_on_date = $2,
    reviews_today = CASE WHEN reviews_on_date = $2 THEN reviews_today + 1 ELSE 1 END,
    updated_at = now()
WHERE id = $1`, userID, day)
	if err != nil {
		return wrap("apply review", err)
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("apply review", err)
	}
	return nil
}

func accessibleDeckSQL(userParam, decksParam string) string {
	return `(d.user_id = ` + userParam + ` OR EXISTS (
    SELECT 1 FROM deck_members m WHERE m.deck_id = d.id AND m.user_id = ` + userParam + `
))
  AND (cardinality(` + decksParam + `::bigint[]) = 0 OR d.id = ANY(` + decksParam + `::bigint[]))`
}

func (r *ReviewRepository) ListDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time) ([]domain.LearnItem, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	q := `
SELECT` + dueItemCols + `
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE r.user_id = $1
  AND r.due_at <= $2
  AND ` + accessibleDeckSQL("$1", "$3") + `
ORDER BY r.due_at ASC, c.id ASC`
	rows, err := r.pool.Query(ctx, q, userID, now, deckIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectLearnItems(rows)
}

func (r *ReviewRepository) ListForLearn(ctx context.Context, userID int64, deckIDs []int64) ([]domain.LearnItem, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	q := `
SELECT` + dueItemCols + `
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE r.user_id = $1
  AND ` + accessibleDeckSQL("$1", "$2") + `
ORDER BY c.id ASC`

	rows, err := r.pool.Query(ctx, q, userID, deckIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectLearnItems(rows)
}

func (r *ReviewRepository) CountDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time) (int, error) {
	if deckIDs == nil {
		deckIDs = []int64{}
	}
	q := `
SELECT count(*)
FROM reviews r
JOIN cards c ON c.id = r.card_id
JOIN decks d ON d.id = c.deck_id
WHERE r.user_id = $1
  AND r.due_at <= $2
  AND ` + accessibleDeckSQL("$1", "$3")
	var n int
	err := r.pool.QueryRow(ctx, q, userID, now, deckIDs).Scan(&n)
	return n, err
}

func collectLearnItems(rows pgx.Rows) ([]domain.LearnItem, error) {
	var out []domain.LearnItem
	for rows.Next() {
		item, err := scanLearnItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
