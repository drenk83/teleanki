package domain

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxDeckNameRunes = 64

type Deck struct {
	ID            int64
	UserID        int64
	Name          string
	ShareCode     string
	OwnerUsername string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (d Deck) OwnedBy(userID int64) bool {
	return d.UserID == userID
}

func (d Deck) ListTitle(viewerID int64) string {
	if d.OwnedBy(viewerID) || d.OwnerUsername == "" {
		return d.Name
	}
	return d.Name + " · @" + d.OwnerUsername
}

const shareAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func NewShareCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i, v := range b {
		out[i] = shareAlphabet[int(v)%len(shareAlphabet)]
	}
	return string(out), nil
}

func NormalizeShareCode(s string) (string, error) {
	code := strings.ToLower(strings.TrimSpace(s))
	if len(code) != 8 {
		return "", fmt.Errorf("invalid share code")
	}
	for _, r := range code {
		if !strings.ContainsRune(shareAlphabet, r) {
			return "", fmt.Errorf("invalid share code")
		}
	}
	return code, nil
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
