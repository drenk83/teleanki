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

func (h *Bot) beginAddCard(ctx context.Context, b *bot.Bot, tgID, chatID, userID, deckID int64) {
	if _, err := h.deckOf(ctx, userID, deckID); err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.sessions.set(tgID, &session{State: stateCardFront, DeckID: deckID})
	h.send(ctx, b, chatID, config.AskCardFront, nil)
}

func (h *Bot) showCardList(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64, page int) {
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
	if len(cards) == 0 {
		h.send(ctx, b, chatID, config.CardEmptyList, kb(row(btn(config.BtnBackToDeck, "d:open:"+strconv.FormatInt(d.ID, 10)))))
		return
	}
	chunk, page, pages := pageSlice(cards, page)
	out := make([][]models.InlineKeyboardButton, 0, len(chunk)+3)
	for _, c := range chunk {
		out = append(out, row(btn(truncate(c.Front, 40), "c:open:"+strconv.FormatInt(c.ID, 10))))
	}
	prefix := "d:cards:" + strconv.FormatInt(d.ID, 10)
	if nav := pager(page, pages, prefix); len(nav) > 0 {
		out = append(out, nav)
	}
	out = append(out, row(btn(config.BtnBackToDeck, "d:open:"+strconv.FormatInt(d.ID, 10))))
	h.send(ctx, b, chatID, fmt.Sprintf(config.CardListTitle, d.Name), kb(out...))
}

func (h *Bot) showCard(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64) {
	c, d, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		slog.Error("Failed to get card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	modeText := config.ModeLabel(c.Mode)
	choicesLine := ""
	if len(c.Choices) > 0 {
		choicesLine = fmt.Sprintf(config.CardChoicesLine, joinChoices(c.Choices))
	}
	text := fmt.Sprintf(config.CardView, clip(c.Front, 1500), clip(c.Back, 1500), modeText, choicesLine)
	id := strconv.FormatInt(c.ID, 10)
	h.send(ctx, b, chatID, text, kb(
		row(btn(config.BtnEditFront, "c:ef:"+id), btn(config.BtnEditBack, "c:eb:"+id)),
		row(btn(config.BtnMode, "c:em:"+id), btn(config.BtnEditChoices, "c:ec:"+id)),
		row(btn(config.BtnDelete, "c:del:"+id)),
		row(btn(config.BtnBackToDeck, "d:open:"+strconv.FormatInt(d.ID, 10))),
	))
}

func (h *Bot) addCardFront(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string) {
	front, err := domain.NormalizeCardText(text)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardFront, nil)
		return
	}
	sess := h.sessions.get(tgID)
	sess.DraftFront = front
	sess.State = stateCardBack
	h.sessions.set(tgID, sess)
	h.send(ctx, b, chatID, config.AskCardBack, nil)
}

func (h *Bot) addCardBack(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string) {
	back, err := domain.NormalizeCardText(text)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardBack, nil)
		return
	}
	sess := h.sessions.get(tgID)
	sess.DraftBack = back
	sess.State = stateCardMode
	h.sessions.set(tgID, sess)
	h.send(ctx, b, chatID, config.AskCardMode, modeKeyboard())
}

func (h *Bot) addCardSetMode(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, mode domain.Mode) {
	sess := h.sessions.get(tgID)
	if _, err := h.deckOf(ctx, userID, sess.DeckID); err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	sess.DraftMode = mode
	if mode == domain.ModeQuiz {
		sess.State = stateCardChoices
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, config.AskCardChoices, nil)
		return
	}
	h.saveNewCard(ctx, b, tgID, chatID, userID, sess, nil)
}

func (h *Bot) addCardChoices(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	choices, err := domain.BuildQuizChoices(sess.DraftBack, splitLines(text))
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskCardChoices, nil)
		return
	}
	h.saveNewCard(ctx, b, tgID, chatID, userID, sess, choices)
}

func (h *Bot) saveNewCard(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, sess *session, choices []string) {
	if choices == nil {
		choices = []string{}
	}
	c := &domain.Card{
		DeckID:  sess.DeckID,
		Front:   sess.DraftFront,
		Back:    sess.DraftBack,
		Mode:    sess.DraftMode,
		Choices: choices,
	}
	created, err := h.cards.Create(ctx, c)
	if err != nil {
		slog.Error("Failed to create card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, created.ID)
}

func (h *Bot) editCardFront(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	front, err := domain.NormalizeCardText(text)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditFront, nil)
		return
	}
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	c.Front = front
	if err := h.cards.Update(ctx, c); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, c.ID)
}

func (h *Bot) editCardBack(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	back, err := domain.NormalizeCardText(text)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditBack, nil)
		return
	}
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	c.Back = back
	if err := h.cards.Update(ctx, c); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, c.ID)
}

func (h *Bot) editCardChoices(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	choices, err := domain.BuildQuizChoices(c.Back, splitLines(text))
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	c.Choices = choices
	if err := h.cards.Update(ctx, c); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, c.ID)
}

func (h *Bot) setCardMode(ctx context.Context, b *bot.Bot, tgID, chatID, userID, cardID int64, mode domain.Mode) {
	c, d, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	c.Mode = mode
	if mode == domain.ModeQuiz {
		if err := domain.ValidateQuizChoices(c.Back, c.Choices); err != nil {
			h.sessions.set(tgID, &session{State: stateEditChoices, CardID: c.ID, DeckID: d.ID})
			if err := h.cards.Update(ctx, c); err != nil {
				slog.Error("Failed to update card", "error", err)
				h.send(ctx, b, chatID, config.TryAgain, nil)
				return
			}
			h.send(ctx, b, chatID, config.AskEditChoices, nil)
			return
		}
	}
	if err := h.cards.Update(ctx, c); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showCard(ctx, b, chatID, userID, c.ID)
}

func (h *Bot) deleteCard(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64) {
	c, d, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if err := h.cards.Delete(ctx, c.ID); err != nil {
		slog.Error("Failed to delete card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func splitLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
