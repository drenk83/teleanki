package storage

import (
	"context"
	"errors"

	"github.com/drenk83/teleanki/internal/domain"
)

var ErrNotFound = errors.New("not found")

type UserRepository interface {
	UpsertByTelegramID(ctx context.Context, telegramID int64, username string) (*domain.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error)
}
