package learn

import (
	"math/rand/v2"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/scheduler"
)

const SessionLimit = 50

type RNG interface {
	IntN(n int) int
	Shuffle(n int, swap func(i, j int))
}

type stdRNG struct{}

func (stdRNG) IntN(n int) int { return rand.IntN(n) }

func (stdRNG) Shuffle(n int, swap func(i, j int)) { rand.Shuffle(n, swap) }

func DefaultRNG() RNG { return stdRNG{} }

type Item struct {
	Card   domain.Card
	Deck   domain.Deck
	Review domain.Review
}

type Session struct {
	Items    []Item
	Index    int
	Shown    bool
	Flipped  bool
	Infinite bool
	Capped   bool
	QuizPerm []int
}

type Kind int

const (
	KindEmpty Kind = iota
	KindLimit
	KindDone
	KindPrompt
	KindReveal
)

type Grade int

const (
	GradeNone Grade = iota
	GradeCorrect
	GradeWrong
)

type View struct {
	Kind     Kind
	Grade    Grade
	Notice   string
	Prompt   string
	Answer   string
	DeckName string
	Index    int
	Total    int
	Mode     domain.Mode
	Choices  []string
}

type Persist struct {
	Review domain.Review
}

func (s Session) Clone() Session {
	cp := s
	if s.Items != nil {
		cp.Items = append([]Item(nil), s.Items...)
	}
	if s.QuizPerm != nil {
		cp.QuizPerm = append([]int(nil), s.QuizPerm...)
	}
	return cp
}

func (s Session) Active() bool {
	if len(s.Items) == 0 {
		return false
	}
	if s.Infinite {
		return true
	}
	return s.Index < len(s.Items)
}

func StartDue(items []Item, remaining int, rng RNG) (Session, View) {
	if remaining <= 0 {
		return Session{}, View{Kind: KindLimit}
	}
	items = showableItems(items)
	limit := SessionLimit
	if remaining < limit {
		limit = remaining
	}
	capped := remaining > SessionLimit && len(items) > SessionLimit
	if len(items) > limit {
		items = items[:limit]
	}
	if len(items) == 0 {
		return Session{}, View{Kind: KindEmpty}
	}
	s := Session{Items: append([]Item(nil), items...), Capped: capped}
	return prompt(s, GradeNone, "", rng)
}

func StartRandom(items []Item, rng RNG) (Session, View) {
	items = showableItems(items)
	if len(items) == 0 {
		return Session{}, View{Kind: KindEmpty}
	}
	s := Session{Items: append([]Item(nil), items...), Infinite: true}
	rng.Shuffle(len(s.Items), func(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] })
	return prompt(s, GradeNone, "", rng)
}

func Show(s Session) (Session, View, bool) {
	if !s.Active() {
		return s, View{}, false
	}
	item := s.Items[s.Index]
	if item.Card.Mode != domain.ModeRecall {
		return s, View{}, false
	}
	promptText, answer := item.Card.PromptAnswer(s.Flipped)
	s.Shown = true
	return s, View{
		Kind:     KindReveal,
		Prompt:   promptText,
		Answer:   answer,
		DeckName: item.Deck.Name,
		Index:    s.Index + 1,
		Total:    len(s.Items),
		Mode:     item.Card.Mode,
	}, true
}

func Rate(s Session, rating scheduler.Rating, now time.Time, grade Grade, notice string, rng RNG) (Session, *Persist, View) {
	if !s.Active() {
		return s, nil, View{Kind: KindDone, Grade: grade, Notice: notice}
	}
	var persist *Persist
	if !s.Infinite {
		item := s.Items[s.Index]
		rev := scheduler.Schedule(item.Review, rating, now)
		rev.CardID = item.Card.ID
		persist = &Persist{Review: rev}
	}
	s, view := advance(s, grade, notice, rng)
	return s, persist, view
}

func Next(s Session, rng RNG) (Session, View) {
	return advance(s, GradeNone, "", rng)
}

func QuizPick(s Session, idx int, now time.Time, rng RNG) (Session, *Persist, View, bool) {
	if !s.Active() || idx < 0 || idx >= len(s.QuizPerm) {
		return s, nil, View{}, false
	}
	item := s.Items[s.Index]
	perm := s.QuizPerm[idx]
	if item.Card.Mode != domain.ModeQuiz || perm < 0 || perm >= len(item.Card.Choices) {
		return s, nil, View{}, false
	}
	choice := item.Card.Choices[perm]
	if domain.MatchTypein(choice, item.Card.Back) {
		s, persist, view := Rate(s, scheduler.RatingGood, now, GradeCorrect, "", rng)
		return s, persist, view, true
	}
	s, persist, view := Rate(s, scheduler.RatingAgain, now, GradeWrong, item.Card.Back, rng)
	return s, persist, view, true
}

func Typein(s Session, text string, now time.Time, rng RNG) (Session, *Persist, View, bool) {
	if !s.Active() {
		return s, nil, View{}, false
	}
	item := s.Items[s.Index]
	if item.Card.Mode != domain.ModeTypein {
		return s, nil, View{}, false
	}
	_, want := item.Card.PromptAnswer(s.Flipped)
	if domain.MatchTypein(text, want) {
		s, persist, view := Rate(s, scheduler.RatingGood, now, GradeCorrect, "", rng)
		return s, persist, view, true
	}
	s, persist, view := Rate(s, scheduler.RatingAgain, now, GradeWrong, want, rng)
	return s, persist, view, true
}

func advance(s Session, grade Grade, notice string, rng RNG) (Session, View) {
	s.Index++
	s.Shown = false
	s.QuizPerm = nil
	s.Flipped = false
	if s.Infinite && len(s.Items) > 0 && s.Index >= len(s.Items) {
		rng.Shuffle(len(s.Items), func(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] })
		s.Index = 0
	}
	return prompt(s, grade, notice, rng)
}

func prompt(s Session, grade Grade, notice string, rng RNG) (Session, View) {
	for s.Index < len(s.Items) {
		item := s.Items[s.Index]
		if item.Card.Mode == domain.ModeQuiz {
			if err := domain.ValidateQuizChoices(item.Card.Back, item.Card.Choices); err != nil {
				s.Index++
				continue
			}
		}
		s.Flipped = item.Card.CanFlip() && rng.IntN(2) == 1
		s.Shown = false
		s.QuizPerm = nil
		p, a := item.Card.PromptAnswer(s.Flipped)
		view := View{
			Kind:     KindPrompt,
			Grade:    grade,
			Notice:   notice,
			Prompt:   p,
			Answer:   a,
			DeckName: item.Deck.Name,
			Index:    s.Index + 1,
			Total:    len(s.Items),
			Mode:     item.Card.Mode,
		}
		if item.Card.Mode == domain.ModeQuiz {
			order, labels := shuffleChoices(item.Card.Choices, rng)
			s.QuizPerm = order
			view.Choices = labels
		}
		return s, view
	}
	if s.Infinite && len(s.Items) > 0 && hasShowable(s.Items) {
		rng.Shuffle(len(s.Items), func(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] })
		s.Index = 0
		return prompt(s, grade, notice, rng)
	}
	return Session{}, View{Kind: KindDone, Grade: grade, Notice: notice}
}

func showableItems(items []Item) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Card.Mode == domain.ModeQuiz {
			if err := domain.ValidateQuizChoices(it.Card.Back, it.Card.Choices); err != nil {
				continue
			}
		}
		out = append(out, it)
	}
	return out
}

func hasShowable(items []Item) bool {
	return len(showableItems(items)) > 0
}

func shuffleChoices(choices []string, rng RNG) ([]int, []string) {
	order := make([]int, len(choices))
	for i := range order {
		order[i] = i
	}
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	labels := make([]string, len(order))
	for i, idx := range order {
		labels[i] = choices[idx]
	}
	return order, labels
}
