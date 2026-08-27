package storage

import (
	"context"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
)

type UserRepository interface {
	UpsertByTelegramID(ctx context.Context, telegramID int64, username string) (*domain.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error)
	SetDailyLimit(ctx context.Context, userID int64, limit int) (*domain.User, error)
	LearnDeckIDs(ctx context.Context, userID int64) ([]int64, error)
	ReplaceLearnDecks(ctx context.Context, userID int64, deckIDs []int64) error
	SetNotify(ctx context.Context, userID int64, enabled bool, hour int) (*domain.User, error)
	SetLearnFree(ctx context.Context, userID int64, free bool) error
	MarkNotified(ctx context.Context, userID int64, day time.Time) error
	ListForNotify(ctx context.Context, hour int, day time.Time) ([]domain.User, error)
}
