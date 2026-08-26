package postgres

import (
	"context"
	"errors"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CardRepository struct {
	pool *pgxpool.Pool
}

func NewCardRepository(db *DB) *CardRepository {
	return &CardRepository{pool: db.Pool}
}

func (r *CardRepository) Create(ctx context.Context, card *domain.Card) (*domain.Card, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	created, err := insertCardWithReview(ctx, tx, card.DeckID, *card)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *CardRepository) CreateMany(ctx context.Context, deckID int64, cards []domain.Card) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, c := range cards {
		if _, err := insertCardWithReview(ctx, tx, deckID, c); err != nil {
			return mapErr(err)
		}
	}
	return tx.Commit(ctx)
}

func (r *CardRepository) GetByID(ctx context.Context, id int64) (*domain.Card, error) {
	const q = `
SELECT id, deck_id, front, back, mode, choices, created_at, updated_at
FROM cards
WHERE id = $1`

	c, err := scanCard(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CardRepository) ListByDeck(ctx context.Context, deckID int64) ([]domain.Card, error) {
	const q = `
SELECT id, deck_id, front, back, mode, choices, created_at, updated_at
FROM cards
WHERE deck_id = $1
ORDER BY id`

	rows, err := r.pool.Query(ctx, q, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CardRepository) CountByDeck(ctx context.Context, deckID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM cards WHERE deck_id = $1`, deckID).Scan(&n)
	return n, err
}

func (r *CardRepository) Update(ctx context.Context, card *domain.Card) error {
	const q = `
UPDATE cards
SET front = $2, back = $3, mode = $4, choices = $5, updated_at = now()
WHERE id = $1
RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q, card.ID, card.Front, card.Back, string(card.Mode), card.Choices).Scan(&card.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	return err
}

func (r *CardRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM cards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}
