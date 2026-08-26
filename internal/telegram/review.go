package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/scheduler"
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
	limit := reviewLimit
	if left < limit {
		limit = left
	}
	items, err := h.reviews.ListDue(ctx, userID, deckIDs, now, limit)
	if err != nil {
		slog.Error("Failed to list due cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if len(items) == 0 {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.ReviewEmpty, nil)
		return
	}
	sess := &session{Review: items, Index: 0}
	h.sessions.set(tgID, sess)
	h.showReviewCard(ctx, b, tgID, chatID, sess, "")
}

func (h *Bot) showReviewCard(ctx context.Context, b *bot.Bot, tgID, chatID int64, sess *session, notice string) {
	if sess.Index >= len(sess.Review) {
		h.sessions.clear(tgID)
		text := config.ReviewDone
		if notice != "" {
			text = notice + "\n\n" + text
		}
		h.send(ctx, b, chatID, text, nil)
		return
	}
	item := sess.Review[sess.Index]
	header := fmt.Sprintf(config.ReviewProgress, sess.Index+1, len(sess.Review), item.Deck.Name)
	front := clip(item.Card.Front, 1500)
	text := header + "\n\n" + front
	if notice != "" {
		text = notice + "\n\n" + text
	}
	mode := item.Card.Mode
	sess.Shown = false
	sess.QuizPerm = nil
	sess.State = stateIdle

	switch mode {
	case domain.ModeQuiz:
		if err := domain.ValidateQuizChoices(item.Card.Back, item.Card.Choices); err == nil {
			order, labels := shuffleChoices(item.Card.Choices)
			sess.QuizPerm = order
			rows := make([][]models.InlineKeyboardButton, 0, len(labels))
			for i, label := range labels {
				rows = append(rows, row(btn(truncate(label, 60), "r:q:"+strconv.Itoa(i))))
			}
			h.sessions.set(tgID, sess)
			h.send(ctx, b, chatID, text, kb(rows...))
			return
		}
		fallthrough
	case domain.ModeRecall:
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text, kb(row(btn(config.BtnShow, "r:show"))))
	case domain.ModeTypein:
		sess.State = stateTypein
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text+"\n\n"+config.TypeinPrompt, nil)
	default:
		h.sessions.set(tgID, sess)
		h.send(ctx, b, chatID, text, kb(row(btn(config.BtnShow, "r:show"))))
	}
}

func (h *Bot) reviewShow(ctx context.Context, b *bot.Bot, tgID, chatID int64) {
	sess := h.sessions.get(tgID)
	if sess.Index >= len(sess.Review) {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	item := sess.Review[sess.Index]
	sess.Shown = true
	h.sessions.set(tgID, sess)
	text := clip(item.Card.Front, 1200) + "\n\n" + clip(item.Card.Back, 1200)
	h.send(ctx, b, chatID, text, ratingKeyboard())
}

func (h *Bot) reviewRate(ctx context.Context, b *bot.Bot, tgID, chatID int64, rating scheduler.Rating) {
	h.reviewRateNotice(ctx, b, tgID, chatID, rating, "")
}

func (h *Bot) reviewRateNotice(ctx context.Context, b *bot.Bot, tgID, chatID int64, rating scheduler.Rating, notice string) {
	sess := h.sessions.get(tgID)
	if sess.Index >= len(sess.Review) {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	item := sess.Review[sess.Index]
	rev := scheduler.Schedule(item.Review, rating, time.Now())
	rev.CardID = item.Card.ID
	if err := h.reviews.Update(ctx, &rev); err != nil {
		slog.Error("Failed to update review", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if u, err := h.users.GetByTelegramID(ctx, tgID); err == nil {
		if _, err := h.users.AddReview(ctx, u.ID, time.Now()); err != nil {
			slog.Error("Failed to count review", "error", err)
		}
	}
	sess.Index++
	sess.Shown = false
	sess.QuizPerm = nil
	sess.State = stateIdle
	h.sessions.set(tgID, sess)
	h.showReviewCard(ctx, b, tgID, chatID, sess, notice)
}

func (h *Bot) reviewQuizPick(ctx context.Context, b *bot.Bot, tgID, chatID int64, idx int) {
	sess := h.sessions.get(tgID)
	if sess.Index >= len(sess.Review) || idx < 0 || idx >= len(sess.QuizPerm) {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	item := sess.Review[sess.Index]
	choice := item.Card.Choices[sess.QuizPerm[idx]]
	if choice == item.Card.Back {
		h.reviewRateNotice(ctx, b, tgID, chatID, scheduler.RatingGood, config.ReviewCorrect)
		return
	}
	h.reviewRateNotice(ctx, b, tgID, chatID, scheduler.RatingAgain, fmt.Sprintf(config.ReviewWrong, item.Card.Back))
}

func (h *Bot) reviewTypein(ctx context.Context, b *bot.Bot, tgID, chatID int64, text string) {
	sess := h.sessions.get(tgID)
	if sess.Index >= len(sess.Review) {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	item := sess.Review[sess.Index]
	if domain.MatchTypein(text, item.Card.Back) {
		h.reviewRateNotice(ctx, b, tgID, chatID, scheduler.RatingGood, config.ReviewCorrect)
		return
	}
	h.reviewRateNotice(ctx, b, tgID, chatID, scheduler.RatingAgain, fmt.Sprintf(config.ReviewWrong, item.Card.Back))
}

func shuffleChoices(choices []string) ([]int, []string) {
	order := make([]int, len(choices))
	for i := range order {
		order[i] = i
	}
	rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	labels := make([]string, len(order))
	for i, idx := range order {
		labels[i] = choices[idx]
	}
	return order, labels
}
