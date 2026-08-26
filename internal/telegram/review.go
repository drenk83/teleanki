package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
	"github.com/drenk83/teleanki/internal/scheduler"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) startReview(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, deckIDs []int64) {
	u, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	now := time.Now()
	left := u.RemainingToday(now)
	if left <= 0 {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, fmt.Sprintf(config.DailyLimitReached, u.DailyLimit), nil)
		return
	}
	items, err := h.reviews.ListDue(ctx, userID, deckIDs, now, 0)
	if err != nil {
		slog.Error("Failed to list due cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	s, view := learn.StartDue(toLearnItems(items), left, learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, u.DailyLimit)
}

func (h *Bot) startRandom(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, deckIDs []int64) {
	items, err := h.reviews.ListForLearn(ctx, userID, deckIDs)
	if err != nil {
		slog.Error("Failed to list cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	s, view := learn.StartRandom(toLearnItems(items), learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, 0)
}

func (h *Bot) reviewShow(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	sess := h.sessions.get(tgID)
	if sess.Learn == nil || !sess.Learn.Active() {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	s, view, ok := learn.Show(*sess.Learn)
	if !ok {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, 0)
}

func (h *Bot) reviewRate(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, rating scheduler.Rating) {
	sess := h.sessions.get(tgID)
	if sess.Learn == nil || !sess.Learn.Active() {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	s, persist, view := learn.Rate(*sess.Learn, rating, time.Now(), learn.GradeNone, "", learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, persist, view, 0)
}

func (h *Bot) reviewNext(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	sess := h.sessions.get(tgID)
	if sess.Learn == nil || !sess.Learn.Active() {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	s, view := learn.Next(*sess.Learn, learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, 0)
}

func (h *Bot) reviewQuizPick(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, idx int) {
	sess := h.sessions.get(tgID)
	if sess.Learn == nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	s, persist, view, ok := learn.QuizPick(*sess.Learn, idx, time.Now(), learn.DefaultRNG())
	if !ok {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.applyLearn(ctx, b, tgID, chatID, userID, s, persist, view, 0)
}

func (h *Bot) reviewTypein(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, text string) {
	sess := h.sessions.get(tgID)
	if sess.Learn == nil {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	s, persist, view, ok := learn.Typein(*sess.Learn, text, time.Now(), learn.DefaultRNG())
	if !ok {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.applyLearn(ctx, b, tgID, chatID, userID, s, persist, view, 0)
}

func (h *Bot) applyLearn(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, s learn.Session, persist *learn.Persist, view learn.View, dailyLimit int) {
	if persist != nil {
		if err := h.reviews.Apply(ctx, &persist.Review, userID, time.Now()); err != nil {
			slog.Error("Failed to apply review", "error", err)
			h.send(ctx, b, chatID, config.TryAgain, nil)
			return
		}
	}
	h.renderLearnView(ctx, b, tgID, chatID, s, view, dailyLimit)
}

func (h *Bot) renderLearnView(ctx context.Context, b *bot.Bot, tgID, chatID int64, s learn.Session, view learn.View, dailyLimit int) {
	notice := gradeNotice(view)
	switch view.Kind {
	case learn.KindEmpty:
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.ReviewEmpty, nil)
		return
	case learn.KindLimit:
		h.sessions.clear(tgID)
		if dailyLimit <= 0 {
			if u, err := h.users.GetByTelegramID(ctx, tgID); err == nil {
				dailyLimit = u.DailyLimit
			}
		}
		h.send(ctx, b, chatID, fmt.Sprintf(config.DailyLimitReached, dailyLimit), nil)
		return
	case learn.KindDone:
		h.sessions.clear(tgID)
		text := h.dueDoneText(ctx, tgID, s)
		if notice != "" {
			text = notice + "\n\n" + text
		}
		h.send(ctx, b, chatID, text, nil)
		return
	}

	sess := &session{Learn: &s}
	header := fmt.Sprintf(config.ReviewProgress, view.Index, view.Total, view.DeckName)
	if s.Infinite {
		header = fmt.Sprintf(config.ReviewRandom, view.DeckName)
	}
	if view.Kind == learn.KindReveal {
		text := clip(view.Prompt, 1200) + "\n\n" + clip(view.Answer, 1200)
		markup := ratingKeyboard()
		if s.Infinite {
			markup = nextKeyboard()
		}
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text, markup)
		return
	}

	text := header + "\n\n" + clip(view.Prompt, 1500)
	if notice != "" {
		text = notice + "\n\n" + text
	}
	switch view.Mode {
	case domain.ModeQuiz:
		rows := make([][]models.InlineKeyboardButton, 0, len(view.Choices))
		for i, label := range view.Choices {
			rows = append(rows, row(btn(truncate(label, 60), "r:q:"+strconv.Itoa(i))))
		}
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text, kb(rows...))
	case domain.ModeTypein:
		sess.State = stateTypein
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text+"\n\n"+config.TypeinPrompt, nil)
	default:
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text, kb(row(btn(config.BtnShow, "r:show"))))
	}
}

func (h *Bot) dueDoneText(ctx context.Context, tgID int64, s learn.Session) string {
	if s.Infinite {
		return config.ReviewCaughtUp
	}
	u, err := h.users.GetByTelegramID(ctx, tgID)
	if err != nil {
		return config.ReviewCaughtUp
	}
	if u.RemainingToday(time.Now()) <= 0 {
		return config.ReviewDone
	}
	if s.Capped {
		return config.ReviewBatchDone
	}
	return config.ReviewCaughtUp
}

func gradeNotice(view learn.View) string {
	switch view.Grade {
	case learn.GradeCorrect:
		return config.ReviewCorrect
	case learn.GradeWrong:
		return fmt.Sprintf(config.ReviewWrong, view.Notice)
	default:
		return ""
	}
}

func toLearnItems(items []storage.DueItem) []learn.Item {
	out := make([]learn.Item, len(items))
	for i, it := range items {
		out[i] = learn.Item{Card: it.Card, Deck: it.Deck, Review: it.Review}
	}
	return out
}
