package storage

import (
	"context"

	"github.com/drenk83/teleanki/internal/domain"
)

type CardRepository interface {
	Create(ctx context.Context, card *domain.Card) (*domain.Card, error)
	CreateMany(ctx context.Context, deckID int64, cards []domain.Card) error
	GetByID(ctx context.Context, id int64) (*domain.Card, error)
	ListByDeck(ctx context.Context, deckID int64) ([]domain.Card, error)
	CountByDeck(ctx context.Context, deckID int64) (int, error)
	Update(ctx context.Context, card *domain.Card) error
	Delete(ctx context.Context, id int64) error
}
