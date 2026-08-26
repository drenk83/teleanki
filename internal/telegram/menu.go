package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/go-telegram/bot"
)

func (h *Bot) showMainMenu(ctx context.Context, b *bot.Bot, tgID, chatID int64) {
	h.sessions.clear(tgID)
	h.send(withNoMenu(ctx), b, chatID, config.MenuTitle, mainMenuKeyboard())
}

func (h *Bot) showHelp(ctx context.Context, b *bot.Bot, chatID int64) {
	h.send(ctx, b, chatID, config.HelpMessage, nil)
}

func (h *Bot) showSettings(ctx context.Context, b *bot.Bot, tgID, chatID int64) {
	u, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	left := u.RemainingToday(time.Now())
	text := fmt.Sprintf(config.SettingsTitle, u.DailyLimit, left) + "\n" + config.SettingsHint
	h.send(ctx, b, chatID, text, settingsKeyboard())
}

func (h *Bot) setDailyLimit(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, limit int) {
	n, err := domain.NormalizeDailyLimit(limit)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidDailyLimit+"\n"+config.AskCustomLimit, nil)
		return
	}
	if _, err := h.users.SetDailyLimit(ctx, userID, n); err != nil {
		slog.Error("Failed to set daily limit", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showSettings(ctx, b, tgID, chatID)
}

func (h *Bot) askCustomLimit(ctx context.Context, b *bot.Bot, tgID, chatID int64) {
	h.sessions.set(tgID, &session{State: stateDailyLimit})
	h.send(ctx, b, chatID, config.AskCustomLimit, nil)
}

func (h *Bot) setDailyLimitFromText(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidDailyLimit+"\n"+config.AskCustomLimit, nil)
		return
	}
	h.setDailyLimit(ctx, b, tgID, chatID, userID, n)
}
