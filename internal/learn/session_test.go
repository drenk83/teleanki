package learn

import (
	"testing"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/scheduler"
)

type stubRNG struct {
	seq    []int
	i      int
	noswap bool
}

func (s *stubRNG) IntN(n int) int {
	if s.i < len(s.seq) {
		v := s.seq[s.i]
		s.i++
		return v % n
	}
	return 0
}

func (s *stubRNG) Shuffle(n int, swap func(i, j int)) {
	if s.noswap {
		return
	}
	for i := n - 1; i > 0; i-- {
		j := s.IntN(i + 1)
		swap(i, j)
	}
}

func recallItem(id int64, front, back string) Item {
	return Item{
		Card:   domain.Card{ID: id, Front: front, Back: back, Mode: domain.ModeRecall, Choices: []string{}},
		Deck:   domain.Deck{Name: "deck"},
		Review: scheduler.NewState(time.Time{}),
	}
}

func quizItem(id int64, back string, choices []string) Item {
	return Item{
		Card:   domain.Card{ID: id, Front: "q", Back: back, Mode: domain.ModeQuiz, Choices: choices},
		Deck:   domain.Deck{Name: "deck"},
		Review: scheduler.NewState(time.Time{}),
	}
}

func typeinItem(id int64, front, back string, reversible bool) Item {
	return Item{
		Card:   domain.Card{ID: id, Front: front, Back: back, Mode: domain.ModeTypein, Reversible: reversible, Choices: []string{}},
		Deck:   domain.Deck{Name: "deck"},
		Review: scheduler.NewState(time.Time{}),
	}
}

func TestStartDueEmptyAndLimit(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	if _, v := StartDue(nil, 10, rng); v.Kind != KindEmpty {
		t.Fatalf("empty items: %v", v.Kind)
	}
	if _, v := StartDue([]Item{recallItem(1, "f", "b")}, 0, rng); v.Kind != KindLimit {
		t.Fatalf("limit: %v", v.Kind)
	}
}

func TestStartDueCapsAtRemaining(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	items := []Item{recallItem(1, "a", "b"), recallItem(2, "c", "d"), recallItem(3, "e", "f")}
	s, v := StartDue(items, 2, rng)
	if v.Kind != KindPrompt || len(s.Items) != 2 {
		t.Fatalf("got kind=%v n=%d", v.Kind, len(s.Items))
	}
	if s.Capped {
		t.Fatal("remaining below session limit is not a batch cap")
	}
}

func TestStartDueMarksCapped(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	items := make([]Item, SessionLimit+1)
	for i := range items {
		items[i] = recallItem(int64(i+1), "f", "b")
	}
	s, v := StartDue(items, SessionLimit+10, rng)
	if v.Kind != KindPrompt || !s.Capped || len(s.Items) != SessionLimit {
		t.Fatalf("capped: kind=%v n=%d capped=%v", v.Kind, len(s.Items), s.Capped)
	}
	s, _ = StartDue(items[:SessionLimit], SessionLimit+10, rng)
	if s.Capped {
		t.Fatal("exactly session limit due is not capped")
	}
	s, _ = StartDue(items[:3], SessionLimit+10, rng)
	if s.Capped {
		t.Fatal("short due queue is not capped")
	}
}

func TestStartDueSkipsInvalidBeforeCap(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	items := make([]Item, SessionLimit+1)
	for i := range items {
		items[i] = quizItem(int64(i+1), "да", []string{})
	}
	items[SessionLimit] = recallItem(99, "f", "b")
	s, v := StartDue(items, SessionLimit+10, rng)
	if v.Kind != KindPrompt || s.Items[s.Index].Card.ID != 99 || s.Capped {
		t.Fatalf("should keep valid after skip: kind=%v id=%d capped=%v", v.Kind, s.Items[s.Index].Card.ID, s.Capped)
	}
}

func TestRandomDoesNotPersist(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	s, v := StartRandom([]Item{recallItem(1, "f", "b")}, rng)
	if v.Kind != KindPrompt || !s.Infinite {
		t.Fatalf("start: %#v %#v", s, v)
	}
	var ok bool
	s, v, ok = Show(s)
	if !ok || v.Kind != KindReveal {
		t.Fatalf("show: ok=%v kind=%v", ok, v.Kind)
	}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, persist, v := Rate(s, scheduler.RatingGood, now, GradeNone, "", rng)
	if persist != nil {
		t.Fatalf("random must not persist: %#v", persist)
	}
	if v.Kind != KindPrompt {
		t.Fatalf("reshuffle: %v", v.Kind)
	}
	if !s.Infinite {
		t.Fatal("still random")
	}
}

func TestDuePersistsAndAgainLeavesSession(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, _ := StartDue([]Item{recallItem(7, "f", "b"), recallItem(8, "c", "d")}, 20, rng)
	s, _, ok := Show(s)
	if !ok {
		t.Fatal("show recall")
	}
	s, persist, v := Rate(s, scheduler.RatingAgain, now, GradeNone, "", rng)
	if persist == nil || persist.Review.CardID != 7 {
		t.Fatalf("persist: %#v", persist)
	}
	if persist.Review.IntervalDays != 1 || persist.Review.Repetitions != 0 {
		t.Fatalf("again: %#v", persist.Review)
	}
	if v.Kind != KindPrompt || s.Items[s.Index].Card.ID != 8 {
		t.Fatalf("next after again: idx=%d id=%d kind=%v", s.Index, s.Items[s.Index].Card.ID, v.Kind)
	}
	if s.Items[len(s.Items)-1].Card.ID != 7 || s.Items[len(s.Items)-1].Review.Repetitions != 0 {
		t.Fatalf("card 7 must sit at end with again state: %#v", s.Items[len(s.Items)-1].Review)
	}
}

func TestQuizGoodAndWrong(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true, seq: []int{0, 0, 0, 0, 0, 0, 0, 0}}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	item := quizItem(1, "да", []string{"нет"})
	s, v := StartDue([]Item{item, recallItem(2, "x", "y")}, 20, rng)
	if v.Kind != KindPrompt || v.Mode != domain.ModeQuiz || len(s.QuizPerm) != 2 {
		t.Fatalf("prompt: %#v perm=%v", v, s.QuizPerm)
	}
	buttons := item.Card.QuizButtons()
	backIdx := -1
	for i, p := range s.QuizPerm {
		if buttons[p] == "да" {
			backIdx = i
		}
	}
	if backIdx < 0 {
		t.Fatal("no correct choice")
	}
	s, persist, v, ok := QuizPick(s, backIdx, now, rng)
	if !ok || persist == nil || persist.Review.Repetitions != 1 {
		t.Fatalf("good: ok=%v persist=%#v", ok, persist)
	}
	if v.Grade != GradeCorrect {
		t.Fatalf("grade=%v", v.Grade)
	}

	s, v = StartDue([]Item{item}, 20, rng)
	wrong := 0
	if s.QuizPerm[0] == 0 {
		wrong = 1
	}
	_, persist, v, ok = QuizPick(s, wrong, now, rng)
	if !ok || persist == nil || persist.Review.Repetitions != 0 {
		t.Fatalf("wrong: ok=%v persist=%#v", ok, persist)
	}
	if v.Grade != GradeWrong || v.Notice != "да" {
		t.Fatalf("wrong view: %#v", v)
	}
}

func TestTypeinMatchesHiddenSide(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{seq: []int{1}, noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, v := StartDue([]Item{typeinItem(1, "hello", "привет", true)}, 20, rng)
	if !s.Flipped || v.Prompt != "привет" {
		t.Fatalf("want flipped prompt, got flip=%v prompt=%q", s.Flipped, v.Prompt)
	}
	_, persist, view, ok := Typein(s, "  Hello ", now, rng)
	if !ok || persist == nil || view.Grade != GradeCorrect {
		t.Fatalf("match hidden: ok=%v persist=%v grade=%v", ok, persist != nil, view.Grade)
	}
	s, _ = StartDue([]Item{typeinItem(1, "hello", "привет", true)}, 20, &stubRNG{seq: []int{1}, noswap: true})
	_, persist, view, ok = Typein(s, "нет", now, &stubRNG{noswap: true})
	if !ok || persist == nil || view.Grade != GradeWrong || view.Notice != "hello" {
		t.Fatalf("mismatch: %#v persist=%v", view, persist != nil)
	}
}

func TestQuizPickFoldMatch(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	item := quizItem(1, "верный", []string{"нет"})
	s, _ := StartDue([]Item{item}, 20, rng)
	buttons := item.Card.QuizButtons()
	idx := -1
	for i, p := range s.QuizPerm {
		if domain.MatchTypein(buttons[p], "верный") {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("no fold choice")
	}
	_, persist, view, ok := QuizPick(s, idx, now, rng)
	if !ok || persist == nil || view.Grade != GradeCorrect {
		t.Fatalf("fold match: ok=%v persist=%v grade=%v", ok, persist != nil, view.Grade)
	}
}

func TestShowOnlyRecall(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	s, _ := StartDue([]Item{quizItem(1, "да", []string{"нет"})}, 20, rng)
	_, _, ok := Show(s)
	if ok {
		t.Fatal("quiz must not reveal")
	}
	s, _ = StartDue([]Item{typeinItem(1, "f", "b", false)}, 20, rng)
	_, _, ok = Show(s)
	if ok {
		t.Fatal("typein must not reveal")
	}
}

func TestQuizNeverFlipped(t *testing.T) {
	t.Parallel()
	item := quizItem(1, "да", []string{"нет"})
	item.Card.Reversible = true
	s, v := StartDue([]Item{item}, 20, &stubRNG{seq: []int{1, 1, 1, 1}, noswap: true})
	if s.Flipped || v.Prompt != "q" {
		t.Fatalf("quiz flip=%v prompt=%q", s.Flipped, v.Prompt)
	}
}

func TestAllInvalidQuizEmpty(t *testing.T) {
	t.Parallel()
	bad := quizItem(1, "да", []string{})
	_, v := StartDue([]Item{bad}, 20, &stubRNG{noswap: true})
	if v.Kind != KindEmpty {
		t.Fatalf("all invalid: %v", v.Kind)
	}
	_, v = StartRandom([]Item{bad}, &stubRNG{noswap: true})
	if v.Kind != KindEmpty {
		t.Fatalf("random invalid: %v", v.Kind)
	}
}

func TestSkipInvalidQuiz(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	bad := quizItem(1, "да", []string{})
	s, v := StartDue([]Item{bad, recallItem(2, "f", "b")}, 20, rng)
	if v.Kind != KindPrompt || s.Items[s.Index].Card.ID != 2 {
		t.Fatalf("should skip bad quiz: idx=%d kind=%v", s.Index, v.Kind)
	}
}

func TestRandomQuizHoldsResult(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	item := quizItem(1, "да", []string{"нет"})
	s, _ := StartRandom([]Item{item, recallItem(2, "x", "y")}, rng)
	buttons := item.Card.QuizButtons()
	idx := 0
	if buttons[s.QuizPerm[0]] != "да" {
		idx = 1
	}
	index := s.Index
	s, persist, v, ok := QuizPick(s, idx, now, rng)
	if !ok || persist != nil || v.Kind != KindResult || v.Grade != GradeCorrect {
		t.Fatalf("result: ok=%v persist=%v kind=%v grade=%v", ok, persist != nil, v.Kind, v.Grade)
	}
	if !s.Shown || s.Index != index || v.Answer != "да" {
		t.Fatalf("must hold: shown=%v idx=%d answer=%q", s.Shown, s.Index, v.Answer)
	}
	if _, _, _, ok = QuizPick(s, idx, now, rng); ok {
		t.Fatal("second pick")
	}
	s, v = Next(s, rng)
	if v.Kind != KindPrompt || s.Items[s.Index].Card.ID != 2 {
		t.Fatalf("next: kind=%v id=%d", v.Kind, s.Items[s.Index].Card.ID)
	}
}

func TestRandomTypeinHoldsResult(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{seq: []int{1}, noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, _ := StartRandom([]Item{typeinItem(1, "hello", "привет", true)}, rng)
	index := s.Index
	s, persist, v, ok := Typein(s, "нет", now, rng)
	if !ok || persist != nil || v.Kind != KindResult || v.Grade != GradeWrong {
		t.Fatalf("result: ok=%v persist=%v kind=%v grade=%v", ok, persist != nil, v.Kind, v.Grade)
	}
	if !s.Shown || s.Index != index || v.Answer != "hello" || v.Notice != "hello" {
		t.Fatalf("must hold: shown=%v idx=%d answer=%q notice=%q", s.Shown, s.Index, v.Answer, v.Notice)
	}
	if _, _, _, ok = Typein(s, "hello", now, rng); ok {
		t.Fatal("second typein")
	}
}

func TestAgainLastWins(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, _ := StartDue([]Item{recallItem(7, "f", "b"), recallItem(8, "c", "d")}, 20, rng)
	s, _, _ = Show(s)
	s, _, _ = Rate(s, scheduler.RatingAgain, now, GradeNone, "", rng)
	s, _, ok := Show(s)
	if !ok || s.Items[s.Index].Card.ID != 8 {
		t.Fatal("card 8")
	}
	s, _, _ = Rate(s, scheduler.RatingGood, now, GradeNone, "", rng)
	if s.Items[s.Index].Card.ID != 7 {
		t.Fatalf("back to 7: id=%d", s.Items[s.Index].Card.ID)
	}
	s, _, _ = Show(s)
	_, persist, _ := Rate(s, scheduler.RatingGood, now, GradeNone, "", rng)
	if persist == nil || persist.Review.Repetitions != 1 {
		t.Fatalf("good after again: %#v", persist)
	}
	if persist.Review.Easiness >= 2.5 {
		t.Fatalf("last rating must start from again EF: %v", persist.Review.Easiness)
	}
}

func TestAgainLastCardReturns(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, _ := StartDue([]Item{recallItem(1, "f", "b")}, 20, rng)
	s, _, _ = Show(s)
	s, persist, v := Rate(s, scheduler.RatingAgain, now, GradeNone, "", rng)
	if persist == nil || v.Kind != KindPrompt || s.Items[s.Index].Card.ID != 1 {
		t.Fatalf("same card: kind=%v id=%d persist=%v", v.Kind, s.Items[s.Index].Card.ID, persist != nil)
	}
}

func TestAgainHitsDailyLimit(t *testing.T) {
	t.Parallel()
	rng := &stubRNG{noswap: true}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	s, _ := StartDue([]Item{recallItem(1, "f", "b"), recallItem(2, "c", "d")}, 1, rng)
	s, _, _ = Show(s)
	_, persist, v := Rate(s, scheduler.RatingAgain, now, GradeNone, "", rng)
	if persist == nil || v.Kind != KindLimit {
		t.Fatalf("limit after last remaining: kind=%v persist=%v", v.Kind, persist != nil)
	}
}
