package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) showDeckList(ctx context.Context, b *bot.Bot, chatID, userID int64, page int) {
	decks, err := h.decks.ListByUser(ctx, userID)
	if err != nil {
		slog.Error("Failed to list decks", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if len(decks) == 0 {
		h.send(ctx, b, chatID, config.DeckEmptyList, kb(row(btn(config.BtnCreateDeck, "d:new"))))
		return
	}
	chunk, page, pages := pageSlice(decks, page)
	out := make([][]models.InlineKeyboardButton, 0, len(chunk)+3)
	for _, d := range chunk {
		out = append(out, row(btn(truncate(d.Name, 40), "d:open:"+strconv.FormatInt(d.ID, 10))))
	}
	if nav := pager(page, pages, "d:list"); len(nav) > 0 {
		out = append(out, nav)
	}
	out = append(out, row(btn(config.BtnCreateDeck, "d:new")))
	h.send(ctx, b, chatID, config.DeckListTitle, kb(out...))
}

func (h *Bot) showDeck(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	d, err := h.deckOf(ctx, userID, deckID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		slog.Error("Failed to get deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	n, err := h.cards.CountByDeck(ctx, d.ID)
	if err != nil {
		slog.Error("Failed to count cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	id := strconv.FormatInt(d.ID, 10)
	text := fmt.Sprintf(config.DeckView, d.Name, n)
	h.send(ctx, b, chatID, text, kb(
		row(btn(config.BtnAddCard, "d:add:"+id), btn(config.BtnCards, "d:cards:"+id)),
		row(btn(config.BtnRename, "d:ren:"+id), btn(config.BtnDelete, "d:del:"+id)),
		row(btn(config.BtnDecks, "d:list:0")),
	))
}

func (h *Bot) createDeckFromText(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, name string) {
	name, err := domain.NormalizeDeckName(name)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidDeckName+"\n"+config.AskDeckName, nil)
		return
	}
	d, err := h.decks.Create(ctx, userID, name)
	if errors.Is(err, storage.ErrAlreadyExists) {
		h.send(ctx, b, chatID, config.DeckNameTaken+"\n"+config.AskDeckName, nil)
		return
	}
	if err != nil {
		slog.Error("Failed to create deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) renameDeckFromText(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, name string) {
	sess := h.sessions.get(tgID)
	name, err := domain.NormalizeDeckName(name)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidDeckName+"\n"+config.AskDeckRename, nil)
		return
	}
	d, err := h.deckOf(ctx, userID, sess.DeckID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	d.Name = name
	err = h.decks.Update(ctx, d)
	if errors.Is(err, storage.ErrAlreadyExists) {
		h.send(ctx, b, chatID, config.DeckNameTaken+"\n"+config.AskDeckRename, nil)
		return
	}
	if err != nil {
		slog.Error("Failed to rename deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) deleteDeck(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	d, err := h.deckOf(ctx, userID, deckID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if err := h.decks.Delete(ctx, d.ID); err != nil {
		slog.Error("Failed to delete deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showDeckList(ctx, b, chatID, userID, 0)
}

func parseID(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

func joinChoices(choices []string) string {
	return strings.Join(choices, " · ")
}
