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

const notifySendTimeout = 5 * time.Second

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
		if err := h.notifyUser(ctx, b, u, now); err != nil {
			slog.Error("Failed to notify user", "error", err, "user_id", u.ID)
		}
	}
}

func (h *Bot) notifyUser(ctx context.Context, b *bot.Bot, u domain.User, now time.Time) error {
	if u.RemainingToday(now) <= 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifySendTimeout)
	defer cancel()
	ids, err := h.users.LearnDeckIDs(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("learn decks: %w", err)
	}
	n, err := h.reviews.CountDue(ctx, u.ID, ids, now)
	if err != nil {
		return fmt.Errorf("count due: %w", err)
	}
	if n <= 0 {
		return nil
	}
	text := fmt.Sprintf(config.NotifyDue, u.RemainingToday(now))
	if err := h.sendErr(withUI(ctx, u.TelegramID, true), b, u.TelegramID, text, kb(
		row(btn(config.BtnNotifyLearn, "menu:learn")),
		row(btn(config.BtnNotifyOff, "s:notify:0"), btn(config.BtnNotifyHour, "s:nhour")),
	)); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	if err := h.users.MarkNotified(ctx, u.ID, now); err != nil {
		return fmt.Errorf("mark notified: %w", err)
	}
	return nil
}
