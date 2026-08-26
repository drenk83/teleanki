package storage

import (
	"context"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
)

type DueItem struct {
	Card   domain.Card
	Deck   domain.Deck
	Review domain.Review
}

type ReviewRepository interface {
	Apply(ctx context.Context, review *domain.Review, userID int64, now time.Time) error
	ListDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time, limit int) ([]DueItem, error)
	ListForLearn(ctx context.Context, userID int64, deckIDs []int64) ([]DueItem, error)
	CountDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time) (int, error)
}
