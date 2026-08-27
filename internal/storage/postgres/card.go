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

func (r *CardRepository) Create(ctx context.Context, userID int64, card *domain.Card) (*domain.Card, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrap("create card", err)
	}
	defer tx.Rollback(ctx)

	created, err := insertCardWithReview(ctx, tx, userID, card.DeckID, *card)
	if err != nil {
		return nil, wrap("create card", mapErr(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrap("create card", err)
	}
	return &created, nil
}

func (r *CardRepository) CreateMany(ctx context.Context, userID, deckID int64, cards []domain.Card) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrap("create cards", err)
	}
	defer tx.Rollback(ctx)

	for _, c := range cards {
		if _, err := insertCardWithReview(ctx, tx, userID, deckID, c); err != nil {
			return wrap("create cards", mapErr(err))
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("create cards", err)
	}
	return nil
}

func (r *CardRepository) GetByID(ctx context.Context, id int64) (*domain.Card, error) {
	const q = `
SELECT ` + cardCols + `
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
SELECT ` + cardCols + `
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

func (r *CardRepository) Update(ctx context.Context, userID int64, card *domain.Card) error {
	const q = `
UPDATE cards c
SET front = $3, back = $4, front_image = $5, back_image = $6, mode = $7, choices = $8, reversible = $9, updated_at = now()
FROM decks d
WHERE c.id = $1 AND c.deck_id = d.id AND d.user_id = $2
RETURNING c.updated_at`

	err := r.pool.QueryRow(ctx, q, card.ID, userID, card.Front, card.Back, card.FrontImage, card.BackImage, string(card.Mode), card.Choices, card.Reversible).Scan(&card.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	return wrap("update card", err)
}

func (r *CardRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `
DELETE FROM cards c
USING decks d
WHERE c.id = $1 AND c.deck_id = d.id AND d.user_id = $2`, id, userID)
	if err != nil {
		return wrap("delete card", err)
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}
