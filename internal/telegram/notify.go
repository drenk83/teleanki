package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/go-telegram/bot"
)

func (h *Bot) notifyLoop(ctx context.Context, b *bot.Bot) {
	h.notifyTick(ctx, b)
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.notifyTick(ctx, b)
		}
	}
}

func (h *Bot) notifyTick(ctx context.Context, b *bot.Bot) {
	now := time.Now()
	hour := now.In(domain.DayLoc).Hour()
	users, err := h.users.ListForNotify(ctx, hour, now)
	if err != nil {
		slog.Error("Failed to list notify users", "error", err)
		return
	}
	for i := range users {
		u := users[i]
		if u.RemainingToday(now) <= 0 {
			continue
		}
		ids, err := h.users.LearnDeckIDs(ctx, u.ID)
		if err != nil {
			slog.Error("Failed to get learn decks", "error", err, "user_id", u.ID)
			continue
		}
		n, err := h.reviews.CountDue(ctx, u.ID, ids, now)
		if err != nil {
			slog.Error("Failed to count due", "error", err, "user_id", u.ID)
			continue
		}
		if n <= 0 {
			continue
		}
		text := fmt.Sprintf(config.NotifyDue, u.RemainingToday(now))
		h.send(withUI(ctx, u.TelegramID, true), b, u.TelegramID, text, kb(
			row(btn(config.BtnNotifyLearn, "menu:learn")),
			row(btn(config.BtnNotifyOff, "s:notify:0"), btn(config.BtnNotifyHour, "s:nhour")),
		))
		if err := h.users.MarkNotified(ctx, u.ID, now); err != nil {
			slog.Error("Failed to mark notified", "error", err, "user_id", u.ID)
		}
	}
}
