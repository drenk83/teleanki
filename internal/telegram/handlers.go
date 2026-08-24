package telegram

import (
	"context"
	"log/slog"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   config.StartMessage,
	})
	if err != nil {
		slog.Error("Failed to send message", "error", err)
		return
	}
	slog.Info("Send message",
		"type_message", "start_message",
		"user_id", update.Message.From.ID,
		"username", update.Message.From.Username,
	)
}

func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   config.HelpMessage,
	})
	if err != nil {
		slog.Error("Failed to send message", "error", err)
		return
	}
	slog.Info("Send message",
		"type_message", "help_message",
		"user_id", update.Message.From.ID,
		"username", update.Message.From.Username,
	)
}

func loggingMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil && update.Message.Text != "" {
			slog.Info("Received message",
				"user_id", update.Message.From.ID,
				"username", update.Message.From.Username,
				"text", update.Message.Text,
			)
		}
		next(ctx, b, update)
	}
}
