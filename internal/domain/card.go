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
	MaxCardTextRunes   = 2000
	MinQuizDistractors = 1
	MaxQuizDistractors = 5
)

type Card struct {
	ID         int64
	DeckID     int64
	Front      string
	Back       string
	FrontImage string
	BackImage  string
	Mode       Mode
	Choices    []string
	Reversible bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func ImageExt(contentType string) (string, error) {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/jpg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", fmt.Errorf("unsupported image type")
	}
}

const MaxImageBytes = 10 << 20

func (c Card) PromptImage(flipped bool) string {
	if flipped && c.CanFlip() {
		return c.BackImage
	}
	return c.FrontImage
}

func (c Card) AnswerImage(flipped bool) string {
	if flipped && c.CanFlip() {
		return c.FrontImage
	}
	return c.BackImage
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

func (c Card) QuizButtons() []string {
	return append([]string{c.Back}, c.Choices...)
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
	choices := make([]string, 0, len(distractors))
	for _, d := range distractors {
		d = strings.TrimSpace(d)
		if d == "" || strings.EqualFold(d, back) || choiceFoldSeen(choices, d) {
			continue
		}
		choices = append(choices, d)
	}
	return choices, ValidateQuizChoices(back, choices)
}

func ValidateQuizChoices(back string, distractors []string) error {
	if len(distractors) < MinQuizDistractors || len(distractors) > MaxQuizDistractors {
		return fmt.Errorf("quiz needs %d–%d distractors", MinQuizDistractors, MaxQuizDistractors)
	}
	kept := make([]string, 0, len(distractors))
	for _, c := range distractors {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("quiz choice must not be empty")
		}
		if strings.EqualFold(c, back) {
			return fmt.Errorf("quiz distractors must not include the back")
		}
		if choiceFoldSeen(kept, c) {
			return fmt.Errorf("quiz choices must be unique")
		}
		kept = append(kept, c)
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
	return strings.EqualFold(PlainCardText(got), PlainCardText(want))
}

func PlainCardText(s string) string {
	s = stripTags(s)
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "*", "")
	return strings.TrimSpace(s)
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' && in {
			in = false
			continue
		}
		if !in {
			b.WriteRune(r)
		}
	}
	return b.String()
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
		if err := ValidateQuizChoices(back, c.Choices); err != nil {
			return Card{}, err
		}
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
