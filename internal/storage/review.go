package storage

import (
	"context"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
)

type ReviewRepository interface {
	Apply(ctx context.Context, review *domain.Review, userID int64, now time.Time) error
	ListDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time) ([]learn.Item, error)
	ListForLearn(ctx context.Context, userID int64, deckIDs []int64) ([]learn.Item, error)
	CountDue(ctx context.Context, userID int64, deckIDs []int64, now time.Time) (int, error)
}
