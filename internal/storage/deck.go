package storage

import (
	"context"

	"github.com/drenk83/teleanki/internal/domain"
)

type DeckRepository interface {
	Create(ctx context.Context, userID int64, name string) (*domain.Deck, error)
	CreateWithCards(ctx context.Context, userID int64, name string, cards []domain.Card) (*domain.Deck, error)
	GetByID(ctx context.Context, id int64) (*domain.Deck, error)
	GetByUserAndName(ctx context.Context, userID int64, name string) (*domain.Deck, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Deck, error)
	Update(ctx context.Context, deck *domain.Deck) error
	Delete(ctx context.Context, id int64) error
}
