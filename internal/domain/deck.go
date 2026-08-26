package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxDeckNameRunes = 64

type Deck struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NormalizeDeckName(s string) (string, error) {
	name := strings.TrimSpace(s)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxDeckNameRunes {
		return "", fmt.Errorf("deck name must be 1–%d characters", MaxDeckNameRunes)
	}
	return name, nil
}

func ConflictDeckName(original string, n int) (string, error) {
	suffix := fmt.Sprintf(" (%d)", n)
	maxBase := MaxDeckNameRunes - utf8.RuneCountInString(suffix)
	if maxBase < 1 {
		return "", fmt.Errorf("deck name must be 1–%d characters", MaxDeckNameRunes)
	}
	base := strings.TrimSpace(original)
	if utf8.RuneCountInString(base) > maxBase {
		base = string([]rune(base)[:maxBase])
		base = strings.TrimSpace(base)
	}
	return NormalizeDeckName(base + suffix)
}
