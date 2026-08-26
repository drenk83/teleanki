package telegram

import (
	"context"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
)

func (h *Bot) handleText(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	switch sess.State {
	case stateDeckName:
		h.createDeckFromText(ctx, b, tgID, chatID, userID, text)
	case stateRenameDeck:
		h.renameDeckFromText(ctx, b, tgID, chatID, userID, text)
	case stateCardFront:
		h.addCardFront(ctx, b, tgID, chatID, text)
	case stateCardBack:
		h.addCardBack(ctx, b, tgID, chatID, text)
	case stateCardChoices:
		h.addCardChoices(ctx, b, tgID, chatID, userID, text)
	case stateEditFront:
		h.editCardFront(ctx, b, tgID, chatID, userID, text)
	case stateEditBack:
		h.editCardBack(ctx, b, tgID, chatID, userID, text)
	case stateEditChoices:
		h.editCardChoices(ctx, b, tgID, chatID, userID, text)
	case stateTypein:
		h.reviewTypein(ctx, b, tgID, chatID, userID, text)
	case stateDailyLimit:
		h.setDailyLimitFromText(ctx, b, tgID, chatID, userID, text)
	case stateCardMode, stateCardReverse, stateImportConflict:
		h.send(ctx, b, chatID, config.UseButtons, nil)
	default:
		h.send(ctx, b, chatID, config.UnknownCommand, nil)
	}
}
