package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrQuizNeedsChoices = errors.New("quiz needs valid choices")

const (
	MaxCardTextRunes = 2000
	MinQuizChoices   = 2
	MaxQuizChoices   = 6
)

type Card struct {
	ID         int64
	DeckID     int64
	Front      string
	Back       string
	Mode       Mode
	Choices    []string
	Reversible bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (c Card) CanFlip() bool {
	return c.Reversible && (c.Mode == ModeRecall || c.Mode == ModeTypein)
}

func (c Card) PromptAnswer(flipped bool) (prompt, answer string) {
	if flipped && c.CanFlip() {
		return c.Back, c.Front
	}
	return c.Front, c.Back
}

func NormalizeCardText(s string) (string, error) {
	text := strings.TrimSpace(s)
	n := utf8.RuneCountInString(text)
	if n < 1 || n > MaxCardTextRunes {
		return "", fmt.Errorf("card text must be 1–%d characters", MaxCardTextRunes)
	}
	return text, nil
}

func BuildQuizChoices(back string, distractors []string) ([]string, error) {
	choices := []string{strings.TrimSpace(back)}
	for _, d := range distractors {
		d = strings.TrimSpace(d)
		if d == "" || choiceFoldSeen(choices, d) {
			continue
		}
		choices = append(choices, d)
	}
	return choices, ValidateQuizChoices(back, choices)
}

func ValidateQuizChoices(back string, choices []string) error {
	if len(choices) < MinQuizChoices || len(choices) > MaxQuizChoices {
		return fmt.Errorf("quiz needs %d–%d choices", MinQuizChoices, MaxQuizChoices)
	}
	kept := make([]string, 0, len(choices))
	hasBack := false
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("quiz choice must not be empty")
		}
		if choiceFoldSeen(kept, c) {
			return fmt.Errorf("quiz choices must be unique")
		}
		kept = append(kept, c)
		if strings.EqualFold(c, back) {
			hasBack = true
		}
	}
	if !hasBack {
		return fmt.Errorf("quiz choices must include the back")
	}
	return nil
}

func choiceFoldSeen(seen []string, s string) bool {
	for _, x := range seen {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

func MatchTypein(got, want string) bool {
	return strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want))
}

func NewCard(deckID int64, front, back string, mode Mode, distractors []string, reversible bool) (Card, error) {
	front, err := NormalizeCardText(front)
	if err != nil {
		return Card{}, err
	}
	back, err = NormalizeCardText(back)
	if err != nil {
		return Card{}, err
	}
	if !mode.Valid() {
		return Card{}, fmt.Errorf("invalid mode %q", mode)
	}
	c := Card{DeckID: deckID, Front: front, Back: back, Mode: mode, Choices: []string{}}
	if mode == ModeQuiz {
		choices, err := BuildQuizChoices(back, distractors)
		if err != nil {
			return Card{}, err
		}
		c.Choices = choices
		return c, nil
	}
	c.Reversible = reversible
	return c, nil
}

func NewCardWithChoices(deckID int64, front, back string, mode Mode, choices []string, reversible bool) (Card, error) {
	front, err := NormalizeCardText(front)
	if err != nil {
		return Card{}, err
	}
	back, err = NormalizeCardText(back)
	if err != nil {
		return Card{}, err
	}
	if !mode.Valid() {
		return Card{}, fmt.Errorf("invalid mode %q", mode)
	}
	c := Card{DeckID: deckID, Front: front, Back: back, Mode: mode, Choices: []string{}}
	if mode == ModeQuiz {
		trimmed := trimChoices(choices)
		if err := ValidateQuizChoices(back, trimmed); err != nil {
			return Card{}, err
		}
		c.Choices = trimmed
		return c, nil
	}
	c.Reversible = reversible
	return c, nil
}

func (c Card) WithFront(front string) (Card, error) {
	front, err := NormalizeCardText(front)
	if err != nil {
		return Card{}, err
	}
	c.Front = front
	return c, nil
}

func (c Card) WithBack(back string) (Card, error) {
	back, err := NormalizeCardText(back)
	if err != nil {
		return Card{}, err
	}
	if c.Mode == ModeQuiz {
		next := cloneStrings(c.Choices)
		for i, ch := range next {
			if strings.EqualFold(ch, c.Back) {
				next[i] = back
			}
		}
		if err := ValidateQuizChoices(back, next); err != nil {
			return Card{}, err
		}
		c.Choices = next
	}
	c.Back = back
	return c, nil
}

func (c Card) WithMode(mode Mode) (Card, error) {
	if !mode.Valid() {
		return Card{}, fmt.Errorf("invalid mode %q", mode)
	}
	if mode == ModeQuiz {
		if err := ValidateQuizChoices(c.Back, c.Choices); err != nil {
			return Card{}, ErrQuizNeedsChoices
		}
		c.Reversible = false
	} else {
		c.Choices = []string{}
	}
	c.Mode = mode
	return c, nil
}

func (c Card) BecomeQuiz(distractors []string) (Card, error) {
	choices, err := BuildQuizChoices(c.Back, distractors)
	if err != nil {
		return Card{}, err
	}
	c.Mode = ModeQuiz
	c.Choices = choices
	c.Reversible = false
	return c, nil
}

func (c Card) BecomeQuizWithBack(back string, distractors []string) (Card, error) {
	back, err := NormalizeCardText(back)
	if err != nil {
		return Card{}, err
	}
	c.Back = back
	return c.BecomeQuiz(distractors)
}

func (c Card) WithReversible(on bool) Card {
	if c.Mode == ModeQuiz {
		c.Reversible = false
		return c
	}
	c.Reversible = on
	return c
}

func (c Card) ToggleReverse() Card {
	return c.WithReversible(!c.Reversible)
}

func trimChoices(choices []string) []string {
	out := make([]string, 0, len(choices))
	for _, c := range choices {
		out = append(out, strings.TrimSpace(c))
	}
	return out
}

func cloneStrings(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return append([]string(nil), s...)
}
