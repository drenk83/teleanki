package postgres

import (
	"context"
	"errors"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeckRepository struct {
	pool *pgxpool.Pool
}

func NewDeckRepository(db *DB) *DeckRepository {
	return &DeckRepository{pool: db.Pool}
}

func (r *DeckRepository) Create(ctx context.Context, userID int64, name string) (*domain.Deck, error) {
	const q = `
INSERT INTO decks (user_id, name)
VALUES ($1, $2)
RETURNING id, user_id, name, created_at, updated_at`

	d, err := scanDeck(r.pool.QueryRow(ctx, q, userID, name))
	if err != nil {
		return nil, mapErr(err)
	}
	return &d, nil
}

func (r *DeckRepository) CreateWithCards(ctx context.Context, userID int64, name string, cards []domain.Card) (*domain.Deck, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const q = `
INSERT INTO decks (user_id, name)
VALUES ($1, $2)
RETURNING id, user_id, name, created_at, updated_at`

	d, err := scanDeck(tx.QueryRow(ctx, q, userID, name))
	if err != nil {
		return nil, mapErr(err)
	}
	for _, c := range cards {
		if _, err := insertCardWithReview(ctx, tx, d.ID, c); err != nil {
			return nil, mapErr(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) GetByID(ctx context.Context, id int64) (*domain.Deck, error) {
	const q = `
SELECT id, user_id, name, created_at, updated_at
FROM decks
WHERE id = $1`

	d, err := scanDeck(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) GetByUserAndName(ctx context.Context, userID int64, name string) (*domain.Deck, error) {
	const q = `
SELECT id, user_id, name, created_at, updated_at
FROM decks
WHERE user_id = $1 AND name = $2`

	d, err := scanDeck(r.pool.QueryRow(ctx, q, userID, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Deck, error) {
	const q = `
SELECT id, user_id, name, created_at, updated_at
FROM decks
WHERE user_id = $1
ORDER BY name`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Deck
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DeckRepository) Update(ctx context.Context, deck *domain.Deck) error {
	const q = `
UPDATE decks
SET name = $2, updated_at = now()
WHERE id = $1
RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q, deck.ID, deck.Name).Scan(&deck.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	return mapErr(err)
}

func (r *DeckRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}
