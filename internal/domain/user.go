package domain

import (
	"fmt"
	"time"
)

const (
	DefaultDailyLimit = 20
	MinDailyLimit     = 1
	MaxDailyLimit     = 200
)

func NormalizeDailyLimit(n int) (int, error) {
	if n < MinDailyLimit || n > MaxDailyLimit {
		return 0, fmt.Errorf("daily limit must be %d–%d", MinDailyLimit, MaxDailyLimit)
	}
	return n, nil
}

type User struct {
	ID            int64
	TelegramID    int64
	Username      string
	DailyLimit    int
	ReviewsToday  int
	ReviewsOnDate time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (u User) RemainingToday(now time.Time) int {
	if !sameUTCDate(u.ReviewsOnDate, now) {
		return u.DailyLimit
	}
	n := u.DailyLimit - u.ReviewsToday
	if n < 0 {
		return 0
	}
	return n
}

func sameUTCDate(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}
