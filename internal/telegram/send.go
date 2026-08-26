package telegram

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type uiKey struct{}

type uiReq struct {
	tgID   int64
	fresh  bool
	noMenu bool
}

func withUI(ctx context.Context, tgID int64, fresh bool) context.Context {
	return context.WithValue(ctx, uiKey{}, uiReq{tgID: tgID, fresh: fresh})
}

func withNoMenu(ctx context.Context) context.Context {
	req, _ := ctx.Value(uiKey{}).(uiReq)
	req.noMenu = true
	return context.WithValue(ctx, uiKey{}, req)
}

func (h *Bot) bindUI(tgID, chatID int64, msgID int) {
	h.uiMu.Lock()
	h.ui[tgID] = uiMsg{chatID: chatID, msgID: msgID}
	h.uiMu.Unlock()
}

func (h *Bot) lastUI(tgID int64) (uiMsg, bool) {
	h.uiMu.Lock()
	defer h.uiMu.Unlock()
	m, ok := h.ui[tgID]
	return m, ok
}

func (h *Bot) send(ctx context.Context, b *bot.Bot, chatID int64, text string, markup models.ReplyMarkup) {
	req, _ := ctx.Value(uiKey{}).(uiReq)
	markup = ensureMenu(markup, req.noMenu)

	if req.tgID != 0 && !req.fresh {
		if h.tryEdit(ctx, b, req.tgID, chatID, text, markup) {
			return
		}
	}

	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	})
	if err != nil {
		slog.Error("Failed to send message", "error", err)
		return
	}
	if req.tgID != 0 {
		h.bindUI(req.tgID, chatID, msg.ID)
	}
}

func (h *Bot) tryEdit(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string, markup models.ReplyMarkup) bool {
	ref, ok := h.lastUI(tgID)
	if !ok || ref.msgID == 0 || ref.chatID != chatID {
		return false
	}
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   ref.msgID,
		Text:        text,
		ReplyMarkup: markup,
	})
	if err == nil || isNotModified(err) {
		return true
	}
	slog.Info("Edit message failed", "error", err)
	return false
}

func (h *Bot) ack(ctx context.Context, b *bot.Bot, id string) {
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: id})
	if err != nil {
		slog.Error("Failed to answer callback", "error", err)
	}
}

func (h *Bot) currentUser(ctx context.Context, telegramID int64, username string) (*domain.User, error) {
	return h.users.UpsertByTelegramID(ctx, telegramID, username)
}

func (h *Bot) deckOf(ctx context.Context, userID, deckID int64) (*domain.Deck, error) {
	d, err := h.decks.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, storage.ErrNotFound
	}
	return d, nil
}

func (h *Bot) cardOf(ctx context.Context, userID, cardID int64) (*domain.Card, *domain.Deck, error) {
	c, err := h.cards.GetByID(ctx, cardID)
	if err != nil {
		return nil, nil, err
	}
	d, err := h.deckOf(ctx, userID, c.DeckID)
	if err != nil {
		return nil, nil, err
	}
	return c, d, nil
}

func ensureMenu(m models.ReplyMarkup, skip bool) models.ReplyMarkup {
	kbMarkup, ok := m.(*models.InlineKeyboardMarkup)
	if !ok || kbMarkup == nil {
		kbMarkup = &models.InlineKeyboardMarkup{}
	}
	if skip {
		return kbMarkup
	}
	for _, r := range kbMarkup.InlineKeyboard {
		for _, b := range r {
			if b.CallbackData == "menu" {
				return kbMarkup
			}
		}
	}
	kbMarkup.InlineKeyboard = append(kbMarkup.InlineKeyboard, row(btn(config.BtnMenu, "menu")))
	return kbMarkup
}

func isNotModified(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "message is not modified")
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
