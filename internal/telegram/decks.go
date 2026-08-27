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
		h.send(ctx, b, chatID, config.DeckEmptyList, kb(
			row(btn(config.BtnCreateDeck, "d:new")),
			row(btn(config.BtnImport, "d:import")),
		))
		return
	}
	chunk, page, pages := pageSlice(decks, page)
	out := make([][]models.InlineKeyboardButton, 0, len(chunk)+4)
	for _, d := range chunk {
		out = append(out, row(btn(truncate(d.ListTitle(userID), 40), "d:open:"+strconv.FormatInt(d.ID, 10))))
	}
	if nav := pager(page, pages, "d:list"); len(nav) > 0 {
		out = append(out, nav)
	}
	out = append(out, row(btn(config.BtnCreateDeck, "d:new"), btn(config.BtnImport, "d:import")))
	from := page*pageSize + 1
	to := from + len(chunk) - 1
	h.send(ctx, b, chatID, fmt.Sprintf(config.DeckListTitle, from, to, len(decks)), kb(out...))
}

func (h *Bot) showDeck(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	h.showDeckNotice(ctx, b, chatID, userID, deckID, "")
}

func (h *Bot) showDeckNotice(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64, notice string) {
	d, err := h.deckSeen(ctx, userID, deckID)
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
	if !d.OwnedBy(userID) {
		owner := d.OwnerUsername
		if owner == "" {
			owner = "—"
		}
		h.send(ctx, b, chatID, fmt.Sprintf(config.ShareMemberView, d.Name, n, owner), kb(
			row(btn(config.BtnCards, "d:cards:"+id)),
			row(btn(config.BtnLeave, "d:leave:"+id)),
			row(btn(config.BtnDecks, "d:list:0")),
		))
		return
	}
	text := notice + fmt.Sprintf(config.DeckView, d.Name, n)
	h.send(ctx, b, chatID, text, kb(
		row(btn(config.BtnAddCard, "d:add:"+id), btn(config.BtnCards, "d:cards:"+id)),
		row(btn(config.BtnRename, "d:ren:"+id), btn(config.BtnDelete, "d:del:"+id)),
		row(btn(config.BtnShare, "d:share:"+id)),
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
	h.showDeckNotice(ctx, b, chatID, userID, d.ID, config.DeckCreated)
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
	err = h.decks.Update(ctx, userID, d)
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
	cards, err := h.cards.ListByDeck(ctx, d.ID)
	if err != nil {
		slog.Error("Failed to list cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if err := h.decks.Delete(ctx, userID, d.ID); err != nil {
		slog.Error("Failed to delete deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	for _, c := range cards {
		h.removeImage(c.FrontImage)
		h.removeImage(c.BackImage)
	}
	h.showDeckList(ctx, b, chatID, userID, 0)
}

func (h *Bot) shareDeck(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	d, err := h.deckOf(ctx, userID, deckID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if d.ShareCode == "" {
		code, err := domain.NewShareCode()
		if err != nil {
			slog.Error("Failed to make share code", "error", err)
			h.send(ctx, b, chatID, config.TryAgain, nil)
			return
		}
		if err := h.decks.SetShareCode(ctx, userID, d.ID, code); err != nil {
			slog.Error("Failed to save share code", "error", err)
			h.send(ctx, b, chatID, config.TryAgain, nil)
			return
		}
		d.ShareCode = code
	}
	id := strconv.FormatInt(d.ID, 10)
	h.send(ctx, b, chatID, fmt.Sprintf(config.ShareShow, d.Name, d.ShareCode), kb(
		row(btn(config.BtnShareRotate, "d:rotate:"+id)),
		row(btn(config.BtnBackToDeck, "d:open:"+id)),
	))
}

func (h *Bot) rotateShare(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	if _, err := h.deckOf(ctx, userID, deckID); err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	code, err := domain.NewShareCode()
	if err != nil {
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if err := h.decks.SetShareCode(ctx, userID, deckID, code); err != nil {
		slog.Error("Failed to rotate share code", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.shareDeck(ctx, b, chatID, userID, deckID)
}

func (h *Bot) beginJoin(ctx context.Context, b *bot.Bot, tgID, chatID int64) {
	h.sessions.set(tgID, &session{State: stateImportWait})
	h.send(ctx, b, chatID, config.ImportWait, nil)
}

func (h *Bot) joinFromText(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	code, err := domain.NormalizeShareCode(text)
	if err != nil {
		h.send(ctx, b, chatID, config.ShareBadCode+"\n"+config.AskShareCode, nil)
		return
	}
	d, err := h.decks.GetByShareCode(ctx, code)
	if errors.Is(err, storage.ErrNotFound) {
		h.send(ctx, b, chatID, config.ShareBadCode+"\n"+config.AskShareCode, nil)
		return
	}
	if err != nil {
		slog.Error("Failed to get deck by share code", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if d.OwnedBy(userID) {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.ShareOwnDeck, nil)
		return
	}
	if err := h.decks.Join(ctx, userID, d.ID); err != nil {
		slog.Error("Failed to join deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.send(ctx, b, chatID, fmt.Sprintf(config.ShareJoined, d.Name), nil)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) leaveDeck(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64) {
	d, err := h.deckSeen(ctx, userID, deckID)
	if err != nil || d.OwnedBy(userID) {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if err := h.decks.Leave(ctx, userID, deckID); err != nil {
		slog.Error("Failed to leave deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.send(ctx, b, chatID, config.ShareLeft, nil)
	h.showDeckList(ctx, b, chatID, userID, 0)
}

func parseID(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

func joinChoices(choices []string) string {
	return strings.Join(choices, " · ")
}
