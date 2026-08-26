package postgres

import (
	"context"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/scheduler"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertCardWithReview(ctx context.Context, q querier, deckID int64, card domain.Card) (domain.Card, error) {
	if card.Choices == nil {
		card.Choices = []string{}
	}
	if card.Mode == domain.ModeQuiz {
		card.Reversible = false
	}
	const cardSQL = `
INSERT INTO cards (deck_id, front, back, mode, choices, reversible)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, deck_id, front, back, mode, choices, reversible, created_at, updated_at`

	var mode string
	err := q.QueryRow(ctx, cardSQL, deckID, card.Front, card.Back, string(card.Mode), card.Choices, card.Reversible).Scan(
		&card.ID,
		&card.DeckID,
		&card.Front,
		&card.Back,
		&mode,
		&card.Choices,
		&card.Reversible,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		return domain.Card{}, err
	}
	card.Mode = domain.Mode(mode)

	st := scheduler.NewState(time.Now())
	const reviewSQL = `
INSERT INTO reviews (card_id, easiness, interval_days, repetitions, due_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now())`
	if _, err := q.Exec(ctx, reviewSQL, card.ID, st.Easiness, st.IntervalDays, st.Repetitions, st.DueAt); err != nil {
		return domain.Card{}, err
	}
	if card.Choices == nil {
		card.Choices = []string{}
	}
	return card, nil
}

func scanCard(row pgx.Row) (domain.Card, error) {
	var c domain.Card
	var mode string
	err := row.Scan(
		&c.ID,
		&c.DeckID,
		&c.Front,
		&c.Back,
		&mode,
		&c.Choices,
		&c.Reversible,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return domain.Card{}, err
	}
	c.Mode = domain.Mode(mode)
	if c.Choices == nil {
		c.Choices = []string{}
	}
	return c, nil
}

func scanDueItem(row pgx.Row) (storage.DueItem, error) {
	var item storage.DueItem
	var cardMode string
	err := row.Scan(
		&item.Card.ID,
		&item.Card.DeckID,
		&item.Card.Front,
		&item.Card.Back,
		&cardMode,
		&item.Card.Choices,
		&item.Card.Reversible,
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
		return storage.DueItem{}, err
	}
	item.Card.Mode = domain.Mode(cardMode)
	if item.Card.Choices == nil {
		item.Card.Choices = []string{}
	}
	return item, nil
}

func scanDeck(row pgx.Row) (domain.Deck, error) {
	var d domain.Deck
	err := row.Scan(
		&d.ID,
		&d.UserID,
		&d.Name,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return domain.Deck{}, err
	}
	return d, nil
}

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.DailyLimit,
		&u.ReviewsToday,
		&u.ReviewsOnDate,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func utcDate(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
