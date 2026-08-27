package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
	"github.com/drenk83/teleanki/internal/scheduler"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	h.ack(ctx, b, q.ID)
	if q.Message.Message == nil {
		return
	}
	chatID := q.Message.Message.Chat.ID
	tgID := q.From.ID
	user, err := h.currentUser(ctx, tgID, q.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	parts := strings.Split(q.Data, ":")
	if len(parts) == 0 {
		return
	}
	switch parts[0] {
	case "menu":
		h.onMenuCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "l":
		h.onLearnCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "s":
		h.onSettingsCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "d":
		h.onDeckCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "c":
		h.onCardCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "m":
		h.onModeCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "v":
		h.onReverseCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "r":
		h.onReviewCallback(ctx, b, tgID, chatID, user.ID, parts)
	case "x":
		h.onCancelCallback(ctx, b, tgID, chatID, user.ID)
	case "i":
		h.onImportCallback(ctx, b, tgID, chatID, user.ID, parts)
	}
}

func (h *Bot) onMenuCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) == 1 {
		h.showMainMenu(ctx, b, tgID, chatID)
		return
	}
	switch parts[1] {
	case "decks":
		h.sessions.clear(tgID)
		h.showDeckList(ctx, b, chatID, userID, 0)
	case "learn":
		h.sessions.clear(tgID)
		h.showLearnSetup(ctx, b, tgID, chatID, userID, 0)
	case "settings":
		h.showSettings(ctx, b, tgID, chatID)
	case "help":
		h.showHelp(ctx, b, chatID)
	}
}

func (h *Bot) onSettingsCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "custom":
		h.askCustomLimit(ctx, b, tgID, chatID)
	case "limit":
		if len(parts) < 3 {
			return
		}
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			return
		}
		h.setDailyLimit(ctx, b, tgID, chatID, userID, n)
	case "notify":
		if len(parts) < 3 {
			return
		}
		h.setNotifyEnabled(ctx, b, tgID, chatID, userID, parts[2] == "1")
	case "nhour":
		h.askNotifyHour(ctx, b, tgID, chatID)
	}
}

func (h *Bot) onLearnCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "toggle":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.toggleLearnDeck(ctx, b, tgID, chatID, userID, id, parsePage(parts, 3))
	case "all":
		h.setLearnAllDecks(ctx, b, tgID, chatID, userID, parsePage(parts, 2))
	case "start":
		h.startLearnFromSetup(ctx, b, tgID, chatID, userID)
	case "mode":
		h.toggleLearnMode(ctx, b, tgID, chatID, userID)
	case "page":
		h.showLearnSetup(ctx, b, tgID, chatID, userID, parsePage(parts, 2))
	}
}

func (h *Bot) onDeckCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "list":
		h.showDeckList(ctx, b, chatID, userID, parsePage(parts, 2))
	case "new":
		h.sessions.set(tgID, &session{State: stateDeckName})
		h.send(ctx, b, chatID, config.AskDeckName, nil)
	case "open":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.showDeck(ctx, b, chatID, userID, id)
	case "import":
		h.beginJoin(ctx, b, tgID, chatID)
	case "share":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.shareDeck(ctx, b, chatID, userID, id)
	case "rotate":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.rotateShare(ctx, b, chatID, userID, id)
	case "leave":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.leaveDeck(ctx, b, chatID, userID, id)
	case "ren":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		if _, err := h.deckOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateRenameDeck, DeckID: id})
		h.send(ctx, b, chatID, config.AskDeckRename, nil)
	case "del":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		d, err := h.deckOf(ctx, userID, id)
		if err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		sid := strconv.FormatInt(id, 10)
		h.send(ctx, b, chatID, fmtDeleteDeck(d.Name), kb(
			row(btn(config.BtnYes, "d:delok:"+sid), btn(config.BtnNo, "d:open:"+sid)),
		))
	case "delok":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.deleteDeck(ctx, b, chatID, userID, id)
	case "add":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.beginAddCard(ctx, b, tgID, chatID, userID, id)
	case "cards":
		id, ok := nthID(parts, 2)
		if !ok {
			return
		}
		h.showCardList(ctx, b, chatID, userID, id, parsePage(parts, 3))
	}
}

func (h *Bot) onCardCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 3 {
		return
	}
	id, ok := nthID(parts, 2)
	if !ok {
		return
	}
	switch parts[1] {
	case "open":
		h.showCard(ctx, b, chatID, userID, id)
	case "ef":
		if _, _, err := h.cardOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateEditFront, CardID: id})
		h.send(ctx, b, chatID, config.AskEditFront, cancelKB())
	case "eb":
		if _, _, err := h.cardOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateEditBack, CardID: id})
		h.send(ctx, b, chatID, config.AskEditBack, cancelKB())
	case "em":
		if _, _, err := h.cardOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		h.send(ctx, b, chatID, config.AskCardMode, cardModeKeyboard(id))
	case "setmode":
		if len(parts) < 4 {
			return
		}
		mode, err := domain.ParseMode(parts[3])
		if err != nil {
			return
		}
		h.setCardMode(ctx, b, tgID, chatID, userID, id, mode)
	case "ec":
		if _, _, err := h.cardOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		h.sessions.set(tgID, &session{State: stateEditChoices, CardID: id})
		h.send(ctx, b, chatID, config.AskEditChoices, cancelKB())
	case "del":
		if _, _, err := h.cardOf(ctx, userID, id); err != nil {
			h.send(ctx, b, chatID, config.SessionExpired, nil)
			return
		}
		sid := strconv.FormatInt(id, 10)
		h.send(ctx, b, chatID, config.ConfirmDeleteCard, kb(
			row(btn(config.BtnYes, "c:delok:"+sid), btn(config.BtnNo, "c:open:"+sid)),
		))
	case "delok":
		h.deleteCard(ctx, b, chatID, userID, id)
	case "rev":
		h.toggleCardReverse(ctx, b, chatID, userID, id)
	case "cf":
		h.clearCardImage(ctx, b, chatID, userID, id, true)
	case "cb":
		h.clearCardImage(ctx, b, chatID, userID, id, false)
	}
}

func (h *Bot) onModeCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	sess := h.sessions.get(tgID)
	mode, err := domain.ParseMode(parts[1])
	if err != nil {
		return
	}
	if sess.State != stateCardMode {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.addCardSetMode(ctx, b, tgID, chatID, userID, mode)
}

func (h *Bot) onReverseCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	h.addCardSetReverse(ctx, b, tgID, chatID, userID, parts[1] == "1")
}

func (h *Bot) onReviewCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	sess := h.sessions.get(tgID)
	now := time.Now()
	rng := learn.DefaultRNG()
	var step learnStep
	switch parts[1] {
	case "show":
		step = stepLearn(sess, learnActShow, 0, 0, "", now, rng)
	case "again":
		step = stepLearn(sess, learnActRate, scheduler.RatingAgain, 0, "", now, rng)
	case "hard":
		step = stepLearn(sess, learnActRate, scheduler.RatingHard, 0, "", now, rng)
	case "good":
		step = stepLearn(sess, learnActRate, scheduler.RatingGood, 0, "", now, rng)
	case "easy":
		step = stepLearn(sess, learnActRate, scheduler.RatingEasy, 0, "", now, rng)
	case "next":
		step = stepLearn(sess, learnActNext, 0, 0, "", now, rng)
	case "q":
		if len(parts) < 3 {
			return
		}
		idx, err := strconv.Atoi(parts[2])
		if err != nil {
			return
		}
		step = stepLearn(sess, learnActQuiz, 0, idx, "", now, rng)
	default:
		return
	}
	h.finishLearnStep(ctx, b, tgID, chatID, userID, step, now)
}

func (h *Bot) onCancelCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	h.cancelWizard(ctx, b, tgID, chatID, userID)
}

func (h *Bot) onImportCallback(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	sess := h.sessions.get(tgID)
	if sess.State != stateImportConflict || sess.Import == nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	switch parts[1] {
	case "new":
		h.finishImportNew(ctx, b, tgID, chatID, userID, sess.Import)
	case "append":
		h.finishImportAppend(ctx, b, tgID, chatID, userID)
	case "cancel":
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.ImportCanceled, nil)
	}
}

func nthID(parts []string, i int) (int64, bool) {
	if i >= len(parts) {
		return 0, false
	}
	return parseID(parts[i])
}

func parsePage(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func fmtDeleteDeck(name string) string {
	return fmt.Sprintf(config.ConfirmDeleteDeck, name)
}
