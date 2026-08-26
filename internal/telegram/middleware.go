package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) loggingMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil && update.Message.From != nil && update.Message.Text != "" {
			slog.Info("Received message",
				"user_id", update.Message.From.ID,
				"username", update.Message.From.Username,
				"text", update.Message.Text,
			)
		}
		if update.CallbackQuery != nil {
			slog.Info("Received callback",
				"user_id", update.CallbackQuery.From.ID,
				"username", update.CallbackQuery.From.Username,
				"data", update.CallbackQuery.Data,
			)
		}
		next(ctx, b, update)
	}
}

func (h *Bot) guardMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil {
			if update.Message.From == nil {
				return
			}
			if update.Message.Chat.Type != models.ChatTypePrivate {
				if strings.HasPrefix(update.Message.Text, "/") {
					h.send(ctx, b, update.Message.Chat.ID, config.PrivateOnly, nil)
				}
				return
			}
			ctx = withUI(ctx, update.Message.From.ID, true)
			if _, err := h.users.UpsertByTelegramID(ctx, update.Message.From.ID, update.Message.From.Username); err != nil {
				slog.Error("Failed to upsert user", "error", err, "telegram_id", update.Message.From.ID)
				h.send(ctx, b, update.Message.Chat.ID, config.TryAgain, nil)
				return
			}
		}
		if update.CallbackQuery != nil {
			msg := update.CallbackQuery.Message.Message
			if msg == nil {
				return
			}
			if msg.Chat.Type != models.ChatTypePrivate {
				h.ack(ctx, b, update.CallbackQuery.ID)
				return
			}
			ctx = withUI(ctx, update.CallbackQuery.From.ID, false)
			h.bindUI(update.CallbackQuery.From.ID, msg.Chat.ID, msg.ID)
			if _, err := h.users.UpsertByTelegramID(ctx, update.CallbackQuery.From.ID, update.CallbackQuery.From.Username); err != nil {
				slog.Error("Failed to upsert user", "error", err, "telegram_id", update.CallbackQuery.From.ID)
				h.ack(ctx, b, update.CallbackQuery.ID)
				h.send(ctx, b, msg.Chat.ID, config.TryAgain, nil)
				return
			}
		}
		next(ctx, b, update)
	}
}
