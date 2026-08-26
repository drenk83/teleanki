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

const cardCols = `id, deck_id, front, back, front_image, back_image, mode, choices, reversible, created_at, updated_at`

const deckCols = `id, user_id, name, COALESCE(share_code, ''), created_at, updated_at`

const userCols = `id, telegram_id, username, daily_limit, reviews_today, reviews_on_date, notify_enabled, notify_hour, notify_on_date, created_at, updated_at`

const dueItemCols = `
    c.id, c.deck_id, c.front, c.back, c.front_image, c.back_image, c.mode, c.choices, c.reversible, c.created_at, c.updated_at,
    d.id, d.user_id, d.name, COALESCE(d.share_code, ''), d.created_at, d.updated_at,
    r.user_id, r.card_id, r.easiness, r.interval_days, r.repetitions, r.due_at, r.updated_at`

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
INSERT INTO cards (deck_id, front, back, front_image, back_image, mode, choices, reversible)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING ` + cardCols

	var mode string
	err := q.QueryRow(ctx, cardSQL, deckID, card.Front, card.Back, card.FrontImage, card.BackImage, string(card.Mode), card.Choices, card.Reversible).Scan(
		&card.ID,
		&card.DeckID,
		&card.Front,
		&card.Back,
		&card.FrontImage,
		&card.BackImage,
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
INSERT INTO reviews (user_id, card_id, easiness, interval_days, repetitions, due_at, updated_at)
SELECT u.uid, $1, $2, $3, $4, $5, now()
FROM (
    SELECT user_id AS uid FROM decks WHERE id = $6
    UNION
    SELECT user_id FROM deck_members WHERE deck_id = $6
) u`
	if _, err := q.Exec(ctx, reviewSQL, card.ID, st.Easiness, st.IntervalDays, st.Repetitions, st.DueAt, deckID); err != nil {
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
		&c.FrontImage,
		&c.BackImage,
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
		&item.Card.FrontImage,
		&item.Card.BackImage,
		&cardMode,
		&item.Card.Choices,
		&item.Card.Reversible,
		&item.Card.CreatedAt,
		&item.Card.UpdatedAt,
		&item.Deck.ID,
		&item.Deck.UserID,
		&item.Deck.Name,
		&item.Deck.ShareCode,
		&item.Deck.CreatedAt,
		&item.Deck.UpdatedAt,
		&item.Review.UserID,
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
		&d.ShareCode,
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
		&u.NotifyEnabled,
		&u.NotifyHour,
		&u.NotifyOnDate,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}
