package telegram

import (
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
)

type Bot struct {
	users storage.UserRepository
}

func CreateBot(tgToken string, users storage.UserRepository) (*bot.Bot, error) {
	h := &Bot{users: users}

	opts := []bot.Option{
		bot.WithMiddlewares(loggingMiddleware),
	}

	b, err := bot.New(tgToken, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, h.startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, h.helpHandler)
	return b, nil
}
