package telegram

import (
	"context"
	"sync"

	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type uiMsg struct {
	chatID int64
	msgID  int
	image  string
}

type Bot struct {
	users    storage.UserRepository
	decks    storage.DeckRepository
	cards    storage.CardRepository
	reviews  storage.ReviewRepository
	images   storage.ImageStore
	sessions *sessionStore
	uiMu     sync.Mutex
	ui       map[int64]uiMsg
}

func CreateBot(
	ctx context.Context,
	tgToken string,
	users storage.UserRepository,
	decks storage.DeckRepository,
	cards storage.CardRepository,
	reviews storage.ReviewRepository,
	images storage.ImageStore,
) (*bot.Bot, error) {
	h := &Bot{
		users:    users,
		decks:    decks,
		cards:    cards,
		reviews:  reviews,
		images:   images,
		sessions: newSessionStore(),
		ui:       make(map[int64]uiMsg),
	}

	opts := []bot.Option{
		bot.WithMiddlewares(h.loggingMiddleware, h.guardMiddleware),
		bot.WithDefaultHandler(h.textHandler),
	}

	b, err := bot.New(tgToken, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, h.startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "menu", bot.MatchTypeCommand, h.menuHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, h.helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "decks", bot.MatchTypeCommand, h.decksHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "newdeck", bot.MatchTypeCommand, h.newDeckHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "learn", bot.MatchTypeCommand, h.learnHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, h.callbackHandler)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Document != nil
	}, h.documentHandler)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && len(update.Message.Photo) > 0
	}, h.photoHandler)
	go h.notifyLoop(ctx, b)
	return b, nil
}
