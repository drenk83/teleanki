package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxCardTextRunes = 2000
	MinQuizChoices   = 2
	MaxQuizChoices   = 6
)

type Card struct {
	ID        int64
	DeckID    int64
	Front     string
	Back      string
	Mode      Mode
	Choices   []string
	CreatedAt time.Time
	UpdatedAt time.Time
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
	seen := map[string]struct{}{back: {}}
	choices := []string{back}
	for _, d := range distractors {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		choices = append(choices, d)
	}
	return choices, ValidateQuizChoices(back, choices)
}

func ValidateQuizChoices(back string, choices []string) error {
	if len(choices) < MinQuizChoices || len(choices) > MaxQuizChoices {
		return fmt.Errorf("quiz needs %d–%d choices", MinQuizChoices, MaxQuizChoices)
	}
	seen := make(map[string]struct{}, len(choices))
	hasBack := false
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("quiz choice must not be empty")
		}
		if _, ok := seen[c]; ok {
			return fmt.Errorf("quiz choices must be unique")
		}
		seen[c] = struct{}{}
		if c == back {
			hasBack = true
		}
	}
	if !hasBack {
		return fmt.Errorf("quiz choices must include the back")
	}
	return nil
}

func MatchTypein(got, want string) bool {
	return strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want))
}
