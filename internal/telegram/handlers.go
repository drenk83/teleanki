package telegram

import (
	"context"
	"log/slog"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	h.sessions.clear(update.Message.From.ID)
	h.send(withNoMenu(ctx), b, update.Message.Chat.ID, config.StartMessage, kb(row(btn(config.BtnOpenMenu, "menu"))))
}

func (h *Bot) menuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	h.showMainMenu(ctx, b, update.Message.From.ID, update.Message.Chat.ID)
}

func (h *Bot) helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	h.showHelp(ctx, b, update.Message.Chat.ID)
}

func (h *Bot) decksHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	user, err := h.currentUser(ctx, update.Message.From.ID, update.Message.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, update.Message.Chat.ID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(update.Message.From.ID)
	h.showDeckList(ctx, b, update.Message.Chat.ID, user.ID, 0)
}

func (h *Bot) newDeckHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	h.sessions.set(update.Message.From.ID, &session{State: stateDeckName})
	h.send(ctx, b, update.Message.Chat.ID, config.AskDeckName, nil)
}

func (h *Bot) learnHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	user, err := h.currentUser(ctx, update.Message.From.ID, update.Message.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, update.Message.Chat.ID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(update.Message.From.ID)
	h.showLearnSetup(ctx, b, update.Message.From.ID, update.Message.Chat.ID, user.ID, 0)
}

func (h *Bot) textHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil || update.Message.Text == "" {
		return
	}
	user, err := h.currentUser(ctx, update.Message.From.ID, update.Message.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, update.Message.Chat.ID, config.TryAgain, nil)
		return
	}
	h.handleText(ctx, b, update.Message.From.ID, update.Message.Chat.ID, user.ID, update.Message.Text)
}
