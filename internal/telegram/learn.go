package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) showLearnSetup(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, page int) {
	u, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	decks, err := h.decks.ListByUser(ctx, userID)
	if err != nil {
		slog.Error("Failed to list decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if len(decks) == 0 {
		h.send(ctx, b, chatID, config.LearnNoneDecks, kb(row(btn(config.BtnCreateDeck, "d:new"))))
		return
	}
	selected, err := h.users.LearnDeckIDs(ctx, userID)
	if err != nil {
		slog.Error("Failed to get learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	all := len(selected) == 0
	selSet := map[int64]struct{}{}
	for _, id := range selected {
		selSet[id] = struct{}{}
	}
	label := config.LearnAllLabel
	if !all {
		names := make([]string, 0, len(selected))
		for _, d := range decks {
			if _, ok := selSet[d.ID]; ok {
				names = append(names, d.Name)
			}
		}
		if len(names) == 0 {
			label = config.LearnAllLabel
			all = true
		} else {
			label = joinChoices(names)
		}
	}
	left := u.RemainingToday(time.Now())
	text := fmt.Sprintf(config.LearnTitle, label, u.DailyLimit, left)

	chunk, page, pages := pageSlice(decks, page)
	rows := make([][]models.InlineKeyboardButton, 0, len(chunk)+4)
	for _, d := range chunk {
		mark := config.LearnMarkOn
		if !all {
			if _, ok := selSet[d.ID]; !ok {
				mark = config.LearnMarkOff
			}
		}
		rows = append(rows, row(btn(mark+truncate(d.Name, 36), fmt.Sprintf("l:toggle:%d:%d", d.ID, page))))
	}
	if nav := pager(page, pages, "l:page"); len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows,
		row(btn(config.BtnLearnAll, "l:all")),
		row(btn(config.BtnLearnStart, "l:start"), btn(config.BtnLearnRandom, "l:random")),
		row(btn(config.BtnMenuSettings, "menu:settings")),
	)
	h.send(ctx, b, chatID, text, kb(rows...))
}

func (h *Bot) toggleLearnDeck(ctx context.Context, b *bot.Bot, tgID, chatID, userID, deckID int64, page int) {
	if _, err := h.deckOf(ctx, userID, deckID); err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	decks, err := h.decks.ListByUser(ctx, userID)
	if err != nil {
		slog.Error("Failed to list decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	selected, err := h.users.LearnDeckIDs(ctx, userID)
	if err != nil {
		slog.Error("Failed to get learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	set := map[int64]struct{}{}
	if len(selected) == 0 {
		for _, d := range decks {
			set[d.ID] = struct{}{}
		}
	} else {
		for _, id := range selected {
			set[id] = struct{}{}
		}
	}
	if _, ok := set[deckID]; ok {
		delete(set, deckID)
	} else {
		set[deckID] = struct{}{}
	}
	var next []int64
	if len(set) > 0 && len(set) < len(decks) {
		next = make([]int64, 0, len(set))
		for id := range set {
			next = append(next, id)
		}
	}
	if err := h.users.ReplaceLearnDecks(ctx, userID, next); err != nil {
		slog.Error("Failed to save learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showLearnSetup(ctx, b, tgID, chatID, userID, page)
}

func (h *Bot) setLearnAllDecks(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	if err := h.users.ReplaceLearnDecks(ctx, userID, nil); err != nil {
		slog.Error("Failed to save learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showLearnSetup(ctx, b, tgID, chatID, userID, 0)
}

func (h *Bot) startLearnFromSetup(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	ids, err := h.users.LearnDeckIDs(ctx, userID)
	if err != nil {
		slog.Error("Failed to get learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.startReview(ctx, b, tgID, chatID, userID, ids)
}

func (h *Bot) startRandomFromSetup(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	ids, err := h.users.LearnDeckIDs(ctx, userID)
	if err != nil {
		slog.Error("Failed to get learn decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.startRandom(ctx, b, tgID, chatID, userID, ids)
}
