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
	h.showCardNotice(ctx, b, chatID, userID, cardID, "")
}

func (h *Bot) showCardNotice(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64, notice string) {
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
	text := notice + fmt.Sprintf(config.CardView, truncate(c.Front, 1500), truncate(c.Back, 1500), modeText, extra)
	id := strconv.FormatInt(c.ID, 10)
	deckID := strconv.FormatInt(d.ID, 10)
	var rows [][]models.InlineKeyboardButton
	if d.OwnedBy(userID) {
		rows = [][]models.InlineKeyboardButton{
			row(btn(config.BtnEditFront, "c:ef:"+id), btn(config.BtnEditBack, "c:eb:"+id)),
		}
		modeRow := row(btn(config.BtnMode, "c:em:"+id))
		if c.Mode == domain.ModeQuiz {
			modeRow = append(modeRow, btn(config.BtnEditChoices, "c:ec:"+id))
		} else {
			modeRow = append(modeRow, btn(config.BtnReverse, "c:rev:"+id))
		}
		rows = append(rows, modeRow)
		rows = append(rows, row(btn(config.BtnAddAnother, "d:add:"+deckID)))
		if c.FrontImage != "" {
			rows = append(rows, row(btn(config.BtnClearFrontPhoto, "c:cf:"+id)))
		}
		if c.BackImage != "" {
			rows = append(rows, row(btn(config.BtnClearBackPhoto, "c:cb:"+id)))
		}
		rows = append(rows, row(btn(config.BtnDelete, "c:del:"+id)))
	}
	rows = append(rows, row(btn(config.BtnBackToDeck, "d:open:"+deckID)))
	h.sendMedia(ctx, b, chatID, text, c.FrontImage, kb(rows...))
}

func (h *Bot) addCardFront(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string) {
	h.setAddCardFront(ctx, b, tgID, chatID, text, "")
}

func (h *Bot) addCardFrontPhoto(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	h.setAddCardFront(ctx, b, tgID, chatID, text, image)
}

func (h *Bot) setAddCardFront(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	sess := h.sessions.get(tgID)
	d, err := sess.Draft.setFront(text, image)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardFront, nil)
		return
	}
	sess.Draft = d
	sess.State = stateCardBack
	h.sessions.set(tgID, sess)
	h.send(ctx, b, chatID, config.AskCardBack, nil)
}

func (h *Bot) addCardBack(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string) {
	h.setAddCardBack(ctx, b, tgID, chatID, text, "")
}

func (h *Bot) addCardBackPhoto(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	h.setAddCardBack(ctx, b, tgID, chatID, text, image)
}

func (h *Bot) setAddCardBack(ctx context.Context, b *bot.Bot, tgID, chatID int64, text, image string) {
	sess := h.sessions.get(tgID)
	d, err := sess.Draft.setBack(text, image)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskCardBack, nil)
		return
	}
	d, next := d.afterBack()
	sess.Draft = d
	sess.State = next
	h.sessions.set(tgID, sess)
	if next == stateCardReverse {
		h.send(ctx, b, chatID, config.AskCardReverse, reverseKeyboard())
		return
	}
	h.send(ctx, b, chatID, config.AskCardMode, modeKeyboard())
}

func (h *Bot) addCardSetMode(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, mode domain.Mode) {
	sess := h.sessions.get(tgID)
	if _, err := h.deckOf(ctx, userID, sess.DeckID); err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if sess.Draft.BackImage != "" && mode != domain.ModeRecall {
		return
	}
	d, next := sess.Draft.setMode(mode)
	sess.Draft = d
	sess.State = next
	h.sessions.set(tgID, sess)
	if next == stateCardChoices {
		h.send(ctx, b, chatID, config.AskCardChoices, nil)
		return
	}
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
	sess.Draft.Reversible = reversible
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
	c, err := sess.Draft.commitNew(sess.DeckID, distractors)
	if err != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskCardChoices, nil)
		return
	}
	created, err := h.cards.Create(ctx, userID, &c)
	if err != nil {
		slog.Error("Failed to create card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showCardNotice(ctx, b, chatID, userID, created.ID, config.CardSaved)
}

func (h *Bot) editCardFront(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	h.commitFrontEdit(ctx, b, tgID, chatID, userID, text, "")
}

func (h *Bot) editCardFrontPhoto(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	h.commitFrontEdit(ctx, b, tgID, chatID, userID, text, image)
}

func (h *Bot) commitFrontEdit(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	sess := h.sessions.get(tgID)
	c, _, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next, err := applyFrontEdit(*c, text, image)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditFront, nil)
		return
	}
	old := ""
	if image != "" {
		old = c.FrontImage
	}
	if err := h.cards.Update(ctx, userID, &next); err != nil {
		if image != "" {
			h.removeImage(image)
		}
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if old != "" {
		h.removeImage(old)
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func (h *Bot) editCardBack(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	h.commitBackEdit(ctx, b, tgID, chatID, userID, text, "")
}

func (h *Bot) editCardBackPhoto(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	h.commitBackEdit(ctx, b, tgID, chatID, userID, text, image)
}

func (h *Bot) commitBackEdit(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text, image string) {
	sess := h.sessions.get(tgID)
	c, d, err := h.cardOf(ctx, userID, sess.CardID)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	edit, err := applyBackEdit(*c, text, image)
	if err != nil {
		if image != "" {
			h.removeImage(image)
		}
		h.send(ctx, b, chatID, config.InvalidCardText+"\n"+config.AskEditBack, nil)
		return
	}
	if edit.NeedChoices {
		if image != "" {
			h.removeImage(image)
		}
		h.sessions.set(tgID, &session{State: stateEditChoices, CardID: c.ID, DeckID: d.ID, Draft: cardDraft{Back: edit.DraftBack}})
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	old := ""
	if image != "" {
		old = c.BackImage
	}
	if err := h.cards.Update(ctx, userID, &edit.Card); err != nil {
		if image != "" {
			h.removeImage(image)
		}
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if old != "" {
		h.removeImage(old)
	}
	h.sessions.clear(tgID)
	h.showCard(ctx, b, chatID, userID, edit.Card.ID)
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
	if sess.Draft.Back != "" {
		next, qerr = c.BecomeQuizWithBack(sess.Draft.Back, splitLines(text))
	} else {
		next, qerr = c.BecomeQuiz(splitLines(text))
	}
	if qerr != nil {
		h.send(ctx, b, chatID, config.InvalidChoices+"\n"+config.AskEditChoices, nil)
		return
	}
	if err := h.cards.Update(ctx, userID, &next); err != nil {
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
	if err := h.cards.Update(ctx, userID, &next); err != nil {
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
	if err := h.cards.Delete(ctx, userID, c.ID); err != nil {
		slog.Error("Failed to delete card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.removeImage(c.FrontImage)
	h.removeImage(c.BackImage)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) toggleCardReverse(ctx context.Context, b *bot.Bot, chatID, userID, cardID int64) {
	c, _, err := h.cardOf(ctx, userID, cardID)
	if err != nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	next := c.ToggleReverse()
	if err := h.cards.Update(ctx, userID, &next); err != nil {
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
	old := c.BackImage
	if front {
		old = c.FrontImage
	}
	next := c.ClearImage(front)
	if err := h.cards.Update(ctx, userID, &next); err != nil {
		slog.Error("Failed to update card", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.removeImage(old)
	h.showCard(ctx, b, chatID, userID, next.ID)
}

func splitLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
