package telegram

import (
	"github.com/go-telegram/bot"
)

func CreateBot(tgToken string) (*bot.Bot, error) {
	opts := []bot.Option{
		bot.WithMiddlewares(loggingMiddleware),
	}

	b, err := bot.New(tgToken, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, helpHandler)
	return b, nil
}
