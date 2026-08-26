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
	h.send(ctx, b, chatID, config.AskCardFront, kb(cancelRow()))
}

func (h *Bot) showCardList(ctx context.Context, b *bot.Bot, chatID, userID, deckID int64, page int) {
	d, err := h.deckSeen(ctx, userID, deckID)
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
		out = append(out, row(btn(truncate(domain.PlainCardText(c.Front), 40), "c:open:"+strconv.FormatInt(c.ID, 10))))
	}
	prefix := "d:cards:" + strconv.FormatInt(d.ID, 10)
	if nav := pager(page, pages, prefix); len(nav) > 0 {
		out = append(out, nav)
	}
	out = append(out, row(btn(config.BtnBackToDeck, "d:open:"+strconv.FormatInt(d.ID, 10))))
	h.send(ctx, b, chatID, fmt.Sprintf(config.CardListTitle, d.Name), kb(out...))
}

func (h *Bot) showCard(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64) {
	c, d, err := h.cardSeen(ctx, userID, cardID)
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
	extra := ""
	if c.Mode == domain.ModeQuiz && len(c.Choices) > 0 {
		extra += fmt.Sprintf(config.CardChoicesLine, joinChoices(c.Choices))
	}
	if c.Mode != domain.ModeQuiz {
		if c.Reversible {
			extra += config.CardReverseOn
		} else {
			extra += config.CardReverseOff
		}
	}
	text := fmt.Sprintf(config.CardView, clip(c.Front, 1500), clip(c.Back, 1500), modeText, extra)
	id := strconv.FormatInt(c.ID, 10)
	var rows [][]models.InlineKeyboardButton
	if d.OwnedBy(userID) {
		rows = [][]models.InlineKeyboardButton{
			row(btn(config.BtnEditFront, "c:ef:"+id), btn(config.BtnEditBack, "c:eb:"+id)),
			row(btn(config.BtnMode, "c:em:"+id)),
		}
		if c.Mode == domain.ModeQuiz {
			rows[1] = append(rows[1], btn(config.BtnEditChoices, "c:ec:"+id))
		}
		if c.Mode != domain.ModeQuiz {
			label := config.BtnReverseOff
			if c.Reversible {
				label = config.BtnReverseOn
			}
			rows = append(rows, row(btn(label, "c:rev:"+id)))
		}
		if c.FrontImage != "" {
			rows = append(rows, row(btn(config.BtnClearFrontPhoto, "c:cf:"+id)))
		}
		if c.BackImage != "" {
			rows = append(rows, row(btn(config.BtnClearBackPhoto, "c:cb:"+id)))
		}
		rows = append(rows, row(btn(config.BtnDelete, "c:del:"+id)))
	}
	rows = append(rows, row(btn(config.BtnBackToDeck, "d:open:"+strconv.FormatInt(d.ID, 10))))
	h.sendMedia(ctx, b, chatID, text, c.FrontImage, kb(rows...))
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
	h.send(ctx, b, chatID, config.AskCardBack, cancelKB())
}

func (h *Bot) addCardFrontPhoto(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	front, err := domain.NormalizeCardText(text)
	if err != nil {
		_ = h.images.Remove(image)
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardFront, nil)
		return
	}
	sess := h.sessions.get(tgID)
	sess.DraftFront = front
	sess.DraftFrontImage = image
	sess.State = stateCardBack
	h.sessions.set(tgID, sess)
	h.send(ctx, b, chatID, config.AskCardBack, cancelKB())
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

func (h *Bot) addCardBackPhoto(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	back, err := domain.NormalizeCardText(text)
	if err != nil {
		_ = h.images.Remove(image)
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardBack, nil)
		return
	}
	sess := h.sessions.get(tgID)
	sess.DraftBack = back
	sess.DraftBackImage = image
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
	sess.DraftReversible = false
	if mode == domain.ModeQuiz {
		sess.State = stateCardChoices
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, config.AskCardChoices, cancelKB())
		return
	}
	sess.State = stateCardReverse
	h.sessions.set(tgID, sess)
	h.send(ctx, b, chatID, config.AskCardReverse, reverseKeyboard())
}

func (h *Bot) addCardSetReverse(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, reversible bool) {
	sess := h.sessions.get(tgID)
	if sess.State != stateCardReverse {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if _, err := h.deckOf(ctx, userID, sess.DeckID); err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	sess.DraftReversible = reversible
	h.saveNewCard(ctx, b, tgID, chatID, userID, sess, nil)
}

func (h *Bot) addCardChoices(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	h.saveNewCard(ctx, b, tgID, chatID, userID, sess, splitLines(text))
}

func (h *Bot) saveNewCard(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, sess *session, distractors []string) {
	if _, err := h.deckOf(ctx, userID, sess.DeckID); err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	c, err := domain.NewCard(sess.DeckID, sess.DraftFront, sess.DraftBack, sess.DraftMode, distractors, sess.DraftReversible)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskCardChoices, cancelKB())
		return
	}
	c.FrontImage = sess.DraftFrontImage
	c.BackImage = sess.DraftBackImage
	created, err := h.cards.Create(ctx, &c)
	if err != nil {
		slog.Error("Failed to create card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, created.ID)
}

func (h *Bot) editCardFront(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := c.WithFront(text)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditFront, nil)
		return
	}
	if err := h.cards.Update(ctx, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) editCardFrontPhoto(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		_ = h.images.Remove(image)
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := c.WithFront(text)
	if err != nil {
		_ = h.images.Remove(image)
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditFront, nil)
		return
	}
	old := next.FrontImage
	next.FrontImage = image
	if err := h.cards.Update(ctx, &next); err != nil {
		_ = h.images.Remove(image)
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	_ = h.images.Remove(old)
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) editCardBack(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	c, d, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := c.WithBack(text)
	if err != nil {
		back, nerr := domain.NormalizeCardText(text)
		if nerr != nil {
			h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditBack, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateEditChoices, CardID: c.ID, DeckID: d.ID, DraftBack: back})
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	if err := h.cards.Update(ctx, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) editCardBackPhoto(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	sess := h.sessions.get(tgID)
	c, d, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		_ = h.images.Remove(image)
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := c.WithBack(text)
	if err != nil {
		_ = h.images.Remove(image)
		back, nerr := domain.NormalizeCardText(text)
		if nerr != nil {
			h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditBack, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateEditChoices, CardID: c.ID, DeckID: d.ID, DraftBack: back})
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	old := next.BackImage
	next.BackImage = image
	if err := h.cards.Update(ctx, &next); err != nil {
		_ = h.images.Remove(image)
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	_ = h.images.Remove(old)
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) editCardChoices(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	var next domain.Card
	var qerr error
	if sess.DraftBack != "" {
		next, qerr = c.BecomeQuizWithBack(sess.DraftBack, splitLines(text))
	} else {
		next, qerr = c.BecomeQuiz(splitLines(text))
	}
	if qerr != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	if err := h.cards.Update(ctx, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) setCardMode(ctx context.Context, b *bot.Bot, tgID, chatID, userID, cardID int64, mode domain.Mode) {
	c, d, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := c.WithMode(mode)
	if errors.Is(err, domain.ErrQuizNeedsChoices) {
		h.sessions.set(tgID, &session{State: stateEditChoices, CardID: c.ID, DeckID: d.ID})
		h.send(ctx, b, chatID, config.AskEditChoices, nil)
		return
	}
	if err != nil {
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if err := h.cards.Update(ctx, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showCard(ctx, b, chatID, userID, next.ID)
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
	_ = h.images.Remove(c.FrontImage)
	_ = h.images.Remove(c.BackImage)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) toggleCardReverse(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64) {
	c, _, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next := c.ToggleReverse()
	if err := h.cards.Update(ctx, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) clearCardImage(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64, front bool) {
	c, _, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	var old string
	if front {
		old = c.FrontImage
		c.FrontImage = ""
	} else {
		old = c.BackImage
		c.BackImage = ""
	}
	if err := h.cards.Update(ctx, c); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	_ = h.images.Remove(old)
	h.showCard(ctx, b, chatID, userID, c.ID)
}

func splitLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
