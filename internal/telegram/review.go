package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
	"github.com/drenk83/teleanki/internal/scheduler"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type learnAction int

const (
	learnActShow learnAction = iota
	learnActRate
	learnActNext
	learnActQuiz
	learnActTypein
)

type learnStep struct {
	Session learn.Session
	Persist *learn.Persist
	View    learn.View
	Expired bool
	Silent  bool
}

type learnScreen struct {
	Clear    bool
	Text     string
	Image    string
	Markup   models.ReplyMarkup
	UseMedia bool
	Sess     *session
}

func stepLearn(sess *session, act learnAction, rating scheduler.Rating, quizIdx int, text string, now time.Time, rng learn.RNG) learnStep {
	if act == learnActTypein {
		if sess == nil || sess.Learn == nil {
			return learnStep{Expired: true}
		}
		s, persist, view, ok := learn.Typein(*sess.Learn, text, now, rng)
		if !ok {
			return learnStep{Expired: true}
		}
		return learnStep{Session: s, Persist: persist, View: view}
	}
	if act == learnActQuiz {
		if sess == nil || sess.Learn == nil {
			return learnStep{Expired: true}
		}
		s, persist, view, ok := learn.QuizPick(*sess.Learn, quizIdx, now, rng)
		if !ok {
			return learnStep{Expired: true}
		}
		return learnStep{Session: s, Persist: persist, View: view}
	}
	if sess == nil || sess.Learn == nil || !sess.Learn.Active() {
		return learnStep{Expired: true}
	}
	switch act {
	case learnActShow:
		s, view, ok := learn.Show(*sess.Learn)
		if !ok {
			return learnStep{Expired: true}
		}
		return learnStep{Session: s, View: view}
	case learnActRate:
		if !sess.Learn.Shown {
			return learnStep{Silent: true}
		}
		s, persist, view := learn.Rate(*sess.Learn, rating, now, learn.GradeNone, "", rng)
		return learnStep{Session: s, Persist: persist, View: view}
	case learnActNext:
		if !sess.Learn.Infinite || !sess.Learn.Shown {
			return learnStep{Silent: true}
		}
		s, view := learn.Next(*sess.Learn, rng)
		return learnStep{Session: s, View: view}
	default:
		return learnStep{Silent: true}
	}
}

func buildLearnScreen(s learn.Session, view learn.View, endText string) learnScreen {
	notice := gradeNotice(view)
	switch view.Kind {
	case learn.KindEmpty:
		return learnScreen{Clear: true, Text: config.ReviewEmpty}
	case learn.KindLimit, learn.KindDone:
		text := endText
		if notice != "" {
			text = notice + "\n\n" + text
		}
		return learnScreen{Clear: true, Text: text}
	}

	sess := &session{Learn: &s}
	header := fmt.Sprintf(config.ReviewProgress, view.Index, view.Total, html.EscapeString(view.DeckName))
	if s.Infinite {
		header = fmt.Sprintf(config.ReviewRandom, html.EscapeString(view.DeckName))
	}
	img := ""
	if s.Active() {
		img = s.Items[s.Index].Card.PromptImage(s.Flipped)
	}
	if view.Kind == learn.KindReveal {
		text := truncate(view.Prompt, 1200) + "\n\n" + truncate(view.Answer, 1200)
		markup := ratingKeyboard()
		if s.Infinite {
			markup = nextKeyboard()
		}
		if s.Active() {
			if a := s.Items[s.Index].Card.AnswerImage(s.Flipped); a != "" {
				img = a
			}
		}
		return learnScreen{Text: text, Image: img, Markup: markup, UseMedia: true, Sess: sess}
	}
	if view.Kind == learn.KindResult {
		text := gradeNotice(view)
		if view.Grade == learn.GradeCorrect && view.Answer != "" {
			if text != "" {
				text += "\n\n"
			}
			text += truncate(view.Answer, 1200)
		}
		return learnScreen{Text: text, Markup: nextKeyboard(), Sess: sess}
	}

	text := header + "\n\n" + truncate(view.Prompt, 1500)
	if notice != "" {
		text = notice + "\n\n" + text
	}
	switch view.Mode {
	case domain.ModeQuiz:
		rows := make([][]models.InlineKeyboardButton, 0, len(view.Choices))
		for i, label := range view.Choices {
			rows = append(rows, row(btn(truncate(domain.PlainCardText(label), 60), "r:q:"+strconv.Itoa(i))))
		}
		return learnScreen{Text: text, Image: img, Markup: kb(rows...), UseMedia: true, Sess: sess}
	case domain.ModeTypein:
		sess.State = stateTypein
		return learnScreen{Text: text + "\n\n" + config.TypeinPrompt, Image: img, UseMedia: true, Sess: sess}
	default:
		return learnScreen{Text: text, Image: img, Markup: kb(row(btn(config.BtnShow, "r:show"))), UseMedia: true, Sess: sess}
	}
}

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
	items, err := h.reviews.ListDue(ctx, userID, deckIDs, now)
	if err != nil {
		slog.Error("Failed to list due cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	s, view := learn.StartDue(items, left, learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, u.DailyLimit, now)
}

func (h *Bot) startRandom(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, deckIDs []int64) {
	now := time.Now()
	items, err := h.reviews.ListForLearn(ctx, userID, deckIDs)
	if err != nil {
		slog.Error("Failed to list cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	s, view := learn.StartRandom(items, learn.DefaultRNG())
	h.applyLearn(ctx, b, tgID, chatID, userID, s, nil, view, 0, now)
}

func (h *Bot) finishLearnStep(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, step learnStep, now time.Time) {
	if step.Silent {
		return
	}
	if step.Expired {
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	h.applyLearn(ctx, b, tgID, chatID, userID, step.Session, step.Persist, step.View, 0, now)
}

func (h *Bot) applyLearn(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, s learn.Session, persist *learn.Persist, view learn.View, dailyLimit int, now time.Time) {
	if persist != nil {
		if err := h.reviews.Apply(ctx, &persist.Review, userID, now); err != nil {
			slog.Error("Failed to apply review", "error", err)
			h.send(ctx, b, chatID, config.TryAgain, nil)
			return
		}
	}
	h.renderLearnView(ctx, b, tgID, chatID, s, view, dailyLimit)
}

func (h *Bot) renderLearnView(ctx context.Context, b *bot.Bot, tgID, chatID int64, s learn.Session, view learn.View, dailyLimit int) {
	endText := ""
	switch view.Kind {
	case learn.KindLimit:
		if dailyLimit <= 0 {
			if u, err := h.users.GetByTelegramID(ctx, tgID); err == nil {
				dailyLimit = u.DailyLimit
			}
		}
		endText = fmt.Sprintf(config.DailyLimitReached, dailyLimit)
	case learn.KindDone:
		endText = h.dueDoneText(ctx, tgID, s)
	}
	screen := buildLearnScreen(s, view, endText)
	if screen.Clear {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, screen.Text, nil)
		return
	}
	h.sessions.set(tgID, screen.Sess)
	if screen.UseMedia {
		h.sendMedia(ctx, b, chatID, screen.Text, screen.Image, screen.Markup)
		return
	}
	h.send(ctx, b, chatID, screen.Text, screen.Markup)
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
