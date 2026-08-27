package telegram

import (
	"fmt"
	"html"
	"testing"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
	"github.com/drenk83/teleanki/internal/scheduler"
	"github.com/go-telegram/bot/models"
)

type fixedRNG struct{}

func (fixedRNG) IntN(n int) int { return 0 }

func (fixedRNG) Shuffle(n int, swap func(i, j int)) {}

func recallSess(shown, infinite, flipped bool) *session {
	s := learn.Session{
		Items: []learn.Item{{
			Card: domain.Card{
				ID: 1, Front: "Q", Back: "A", Mode: domain.ModeRecall,
				FrontImage: "front.jpg", BackImage: "back.jpg",
			},
			Deck:   domain.Deck{Name: "Deck"},
			Review: scheduler.NewState(time.Time{}),
		}},
		Shown:    shown,
		Infinite: infinite,
		Flipped:  flipped,
	}
	return &session{Learn: &s}
}

func quizSess() *session {
	s := learn.Session{
		Items: []learn.Item{{
			Card: domain.Card{
				ID: 2, Front: "Q", Back: "yes", Mode: domain.ModeQuiz,
				Choices: []string{"no"},
			},
			Deck:   domain.Deck{Name: "Deck"},
			Review: scheduler.NewState(time.Time{}),
		}},
		QuizPerm: []int{0, 1},
	}
	return &session{Learn: &s}
}

func typeinSess() *session {
	s := learn.Session{
		Items: []learn.Item{{
			Card:   domain.Card{ID: 3, Front: "Q", Back: "Ans", Mode: domain.ModeTypein},
			Deck:   domain.Deck{Name: "Deck"},
			Review: scheduler.NewState(time.Time{}),
		}},
	}
	return &session{Learn: &s}
}

func callbackData(m models.ReplyMarkup) [][]string {
	kb, ok := m.(*models.InlineKeyboardMarkup)
	if !ok || kb == nil {
		return nil
	}
	out := make([][]string, 0, len(kb.InlineKeyboard))
	for _, r := range kb.InlineKeyboard {
		row := make([]string, 0, len(r))
		for _, b := range r {
			row = append(row, b.CallbackData)
		}
		out = append(out, row)
	}
	return out
}

func TestBuildLearnScreenTerminal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		view    learn.View
		endText string
		want    string
	}{
		{"empty", learn.View{Kind: learn.KindEmpty, Grade: learn.GradeCorrect}, "ignored", config.ReviewEmpty},
		{"limit", learn.View{Kind: learn.KindLimit}, "LIM", "LIM"},
		{"limit notice", learn.View{Kind: learn.KindLimit, Grade: learn.GradeWrong, Notice: "A"}, "LIM", fmt.Sprintf(config.ReviewWrong, "A") + "\n\nLIM"},
		{"done", learn.View{Kind: learn.KindDone}, "DONE", "DONE"},
		{"done correct", learn.View{Kind: learn.KindDone, Grade: learn.GradeCorrect}, "DONE", config.ReviewCorrect + "\n\nDONE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildLearnScreen(learn.Session{}, tt.view, tt.endText)
			if !got.Clear || got.Text != tt.want || got.Sess != nil {
				t.Fatalf("clear=%v text=%q sess=%v", got.Clear, got.Text, got.Sess != nil)
			}
		})
	}
}

func TestBuildLearnScreenPromptRecall(t *testing.T) {
	t.Parallel()
	sess := recallSess(false, false, false)
	view := learn.View{
		Kind: learn.KindPrompt, Prompt: "Q", Answer: "A",
		DeckName: "Deck", Index: 1, Total: 1, Mode: domain.ModeRecall,
	}
	got := buildLearnScreen(*sess.Learn, view, "")
	want := fmt.Sprintf(config.ReviewProgress, 1, 1, html.EscapeString("Deck")) + "\n\nQ"
	if got.Clear || got.Text != want || got.Image != "front.jpg" || !got.UseMedia {
		t.Fatalf("text=%q img=%q media=%v clear=%v", got.Text, got.Image, got.UseMedia, got.Clear)
	}
	if got.Sess == nil || got.Sess.State != "" || got.Sess.Learn == nil {
		t.Fatal("session")
	}
	if d := callbackData(got.Markup); len(d) != 1 || d[0][0] != "r:show" {
		t.Fatalf("markup %#v", d)
	}
}

func TestBuildLearnScreenPromptInfinite(t *testing.T) {
	t.Parallel()
	sess := recallSess(false, true, false)
	view := learn.View{
		Kind: learn.KindPrompt, Prompt: "Q", DeckName: "Deck",
		Index: 1, Total: 1, Mode: domain.ModeRecall,
	}
	got := buildLearnScreen(*sess.Learn, view, "")
	want := fmt.Sprintf(config.ReviewRandom, html.EscapeString("Deck")) + "\n\nQ"
	if got.Text != want {
		t.Fatalf("got %q", got.Text)
	}
}

func TestBuildLearnScreenReveal(t *testing.T) {
	t.Parallel()
	sess := recallSess(true, false, false)
	view := learn.View{Kind: learn.KindReveal, Prompt: "Q", Answer: "A"}
	got := buildLearnScreen(*sess.Learn, view, "")
	if got.Text != "Q\n\nA" || got.Image != "back.jpg" || !got.UseMedia {
		t.Fatalf("text=%q img=%q", got.Text, got.Image)
	}
	if d := callbackData(got.Markup); len(d) != 1 || d[0][0] != "r:again" || d[0][3] != "r:easy" {
		t.Fatalf("markup %#v", d)
	}
}

func TestBuildLearnScreenRevealInfinite(t *testing.T) {
	t.Parallel()
	sess := recallSess(true, true, false)
	view := learn.View{Kind: learn.KindReveal, Prompt: "Q", Answer: "A"}
	got := buildLearnScreen(*sess.Learn, view, "")
	if d := callbackData(got.Markup); len(d) != 1 || d[0][0] != "r:next" {
		t.Fatalf("markup %#v", d)
	}
}

func TestBuildLearnScreenQuiz(t *testing.T) {
	t.Parallel()
	s := quizSess().Learn
	view := learn.View{
		Kind: learn.KindPrompt, Prompt: "Q", DeckName: "Deck",
		Index: 1, Total: 1, Mode: domain.ModeQuiz, Choices: []string{"yes", "no"},
	}
	got := buildLearnScreen(*s, view, "")
	d := callbackData(got.Markup)
	if len(d) != 2 || d[0][0] != "r:q:0" || d[1][0] != "r:q:1" {
		t.Fatalf("markup %#v", d)
	}
}

func TestBuildLearnScreenTypein(t *testing.T) {
	t.Parallel()
	s := typeinSess().Learn
	view := learn.View{
		Kind: learn.KindPrompt, Prompt: "Q", DeckName: "Deck",
		Index: 1, Total: 1, Mode: domain.ModeTypein,
	}
	got := buildLearnScreen(*s, view, "")
	want := fmt.Sprintf(config.ReviewProgress, 1, 1, html.EscapeString("Deck")) + "\n\nQ\n\n" + config.TypeinPrompt
	if got.Text != want || got.Sess == nil || got.Sess.State != stateTypein || got.Markup != nil {
		t.Fatalf("text=%q state=%q markup=%v", got.Text, got.Sess.State, got.Markup)
	}
}

func TestBuildLearnScreenResult(t *testing.T) {
	t.Parallel()
	s := typeinSess().Learn
	s.Shown = true
	got := buildLearnScreen(*s, learn.View{Kind: learn.KindResult, Grade: learn.GradeCorrect, Answer: "Ans"}, "")
	if got.UseMedia || got.Text != config.ReviewCorrect+"\n\nAns" {
		t.Fatalf("text=%q media=%v", got.Text, got.UseMedia)
	}
	if d := callbackData(got.Markup); len(d) != 1 || d[0][0] != "r:next" {
		t.Fatalf("markup %#v", d)
	}
	wrong := buildLearnScreen(*s, learn.View{Kind: learn.KindResult, Grade: learn.GradeWrong, Notice: "Ans"}, "")
	if wrong.Text != fmt.Sprintf(config.ReviewWrong, "Ans") {
		t.Fatalf("wrong %q", wrong.Text)
	}
}

func TestStepLearn(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	rng := fixedRNG{}
	t.Run("nil expired", func(t *testing.T) {
		t.Parallel()
		if got := stepLearn(nil, learnActShow, 0, 0, "", now, rng); !got.Expired {
			t.Fatal(got)
		}
	})
	t.Run("show", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(recallSess(false, false, false), learnActShow, 0, 0, "", now, rng)
		if got.Expired || got.Silent || got.View.Kind != learn.KindReveal || !got.Session.Shown {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("rate silent", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(recallSess(false, false, false), learnActRate, scheduler.RatingGood, 0, "", now, rng)
		if !got.Silent || got.Expired {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("rate", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(recallSess(true, false, false), learnActRate, scheduler.RatingGood, 0, "", now, rng)
		if got.Expired || got.Silent || got.Persist == nil {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("next silent", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(recallSess(true, false, false), learnActNext, 0, 0, "", now, rng)
		if !got.Silent {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("next", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(recallSess(true, true, false), learnActNext, 0, 0, "", now, rng)
		if got.Expired || got.Silent || got.Persist != nil {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("quiz", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(quizSess(), learnActQuiz, 0, 0, "", now, rng)
		if got.Expired || got.Silent || got.View.Grade != learn.GradeCorrect {
			t.Fatalf("%#v kind=%v grade=%v", got, got.View.Kind, got.View.Grade)
		}
	})
	t.Run("quiz bad idx", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(quizSess(), learnActQuiz, 0, 9, "", now, rng)
		if !got.Expired {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("typein", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(typeinSess(), learnActTypein, 0, 0, "ans", now, rng)
		if got.Expired || got.View.Grade != learn.GradeCorrect {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("typein nil", func(t *testing.T) {
		t.Parallel()
		got := stepLearn(&session{}, learnActTypein, 0, 0, "ans", now, rng)
		if !got.Expired {
			t.Fatalf("%#v", got)
		}
	})
}
