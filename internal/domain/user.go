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

const (
	DefaultNotifyHour = 19
	MinNotifyHour     = 0
	MaxNotifyHour     = 23
)

func NormalizeNotifyHour(n int) (int, error) {
	if n < MinNotifyHour || n > MaxNotifyHour {
		return 0, fmt.Errorf("notify hour must be %d–%d", MinNotifyHour, MaxNotifyHour)
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
	NotifyEnabled bool
	NotifyHour    int
	NotifyOnDate  time.Time
	LearnFree     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (u User) RemainingToday(now time.Time) int {
	if !SameDay(u.ReviewsOnDate, now) {
		return u.DailyLimit
	}
	n := u.DailyLimit - u.ReviewsToday
	if n < 0 {
		return 0
	}
	return n
}

var DayLoc = time.FixedZone("MSK", 3*60*60)

func SameDay(a, b time.Time) bool {
	ay, am, ad := a.In(DayLoc).Date()
	by, bm, bd := b.In(DayLoc).Date()
	return ay == by && am == bm && ad == bd
}

func DayDate(t time.Time) time.Time {
	y, m, d := t.In(DayLoc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
