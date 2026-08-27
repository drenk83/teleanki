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
RETURNING ` + deckCols

	d, err := scanDeck(r.pool.QueryRow(ctx, q, userID, name))
	if err != nil {
		return nil, wrap("create deck", mapErr(err))
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
RETURNING ` + deckCols

	d, err := scanDeck(tx.QueryRow(ctx, q, userID, name))
	if err != nil {
		return nil, mapErr(err)
	}
	for _, c := range cards {
		if _, err := insertCardWithReview(ctx, tx, userID, d.ID, c); err != nil {
			return nil, wrap("import deck", mapErr(err))
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) GetByID(ctx context.Context, id int64) (*domain.Deck, error) {
	const q = `
SELECT d.id, d.user_id, d.name, COALESCE(d.share_code, ''), d.created_at, d.updated_at, u.username
FROM decks d
JOIN users u ON u.id = d.user_id
WHERE d.id = $1`

	var d domain.Deck
	err := r.pool.QueryRow(ctx, q, id).Scan(&d.ID, &d.UserID, &d.Name, &d.ShareCode, &d.CreatedAt, &d.UpdatedAt, &d.OwnerUsername)
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
SELECT ` + deckCols + `
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
SELECT d.id, d.user_id, d.name, COALESCE(d.share_code, ''), d.created_at, d.updated_at, u.username
FROM decks d
JOIN users u ON u.id = d.user_id
WHERE d.user_id = $1
   OR EXISTS (SELECT 1 FROM deck_members m WHERE m.deck_id = d.id AND m.user_id = $1)
ORDER BY (d.user_id = $1) DESC, d.name`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Deck
	for rows.Next() {
		var d domain.Deck
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.ShareCode, &d.CreatedAt, &d.UpdatedAt, &d.OwnerUsername); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DeckRepository) GetByShareCode(ctx context.Context, code string) (*domain.Deck, error) {
	const q = `
SELECT ` + deckCols + `
FROM decks
WHERE share_code = $1`
	d, err := scanDeck(r.pool.QueryRow(ctx, q, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) SetShareCode(ctx context.Context, userID, deckID int64, code string) error {
	var arg any
	if code == "" {
		arg = nil
	} else {
		arg = code
	}
	tag, err := r.pool.Exec(ctx, `UPDATE decks SET share_code = $2, updated_at = now() WHERE id = $1 AND user_id = $3`, deckID, arg, userID)
	if err != nil {
		return wrap("set share code", mapErr(err))
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) Join(ctx context.Context, userID, deckID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
INSERT INTO deck_members (deck_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`, deckID, userID); err != nil {
		return err
	}
	st := newReviewState()
	if _, err := tx.Exec(ctx, `
INSERT INTO reviews (user_id, card_id, easiness, interval_days, repetitions, due_at, updated_at)
SELECT $2, c.id, $3, $4, $5, $6, now()
FROM cards c
WHERE c.deck_id = $1
ON CONFLICT (user_id, card_id) DO NOTHING`, deckID, userID, st.Easiness, st.IntervalDays, st.Repetitions, st.DueAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DeckRepository) Leave(ctx context.Context, userID, deckID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM deck_members WHERE deck_id = $1 AND user_id = $2`, deckID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *DeckRepository) IsMember(ctx context.Context, userID, deckID int64) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM deck_members WHERE deck_id = $1 AND user_id = $2`, deckID, userID).Scan(&n)
	return n > 0, err
}

func (r *DeckRepository) Update(ctx context.Context, userID int64, deck *domain.Deck) error {
	const q = `
UPDATE decks
SET name = $2, updated_at = now()
WHERE id = $1 AND user_id = $3
RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q, deck.ID, deck.Name, userID).Scan(&deck.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.ErrNotFound
	}
	return wrap("update deck", mapErr(err))
}

func (r *DeckRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM decks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return wrap("delete deck", err)
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}
	return nil
}
