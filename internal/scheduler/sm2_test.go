package scheduler

import (
	"math"
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewState(now)
	if s.Easiness != 2.5 || s.IntervalDays != 0 || s.Repetitions != 0 || !s.DueAt.Equal(now) {
		t.Fatalf("%+v", s)
	}
}

func TestGoodChain(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewState(now)

	s = Schedule(s, RatingGood, now)
	if s.Repetitions != 1 || s.IntervalDays != 1 || s.Easiness != 2.5 {
		t.Fatalf("first good: %+v", s)
	}
	if !s.DueAt.Equal(now.AddDate(0, 0, 1)) {
		t.Fatalf("first due %v", s.DueAt)
	}

	s = Schedule(s, RatingGood, now)
	if s.Repetitions != 2 || s.IntervalDays != 6 {
		t.Fatalf("second good: %+v", s)
	}

	s = Schedule(s, RatingGood, now)
	want := int(math.Round(6 * 2.5))
	if s.Repetitions != 3 || s.IntervalDays != want {
		t.Fatalf("third good: %+v want interval %d", s, want)
	}
}

func TestAgainResets(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewState(now)
	s = Schedule(s, RatingGood, now)
	s = Schedule(s, RatingGood, now)
	beforeEF := s.Easiness
	s = Schedule(s, RatingAgain, now)
	if s.Repetitions != 0 || s.IntervalDays != 1 {
		t.Fatalf("again: %+v", s)
	}
	if !s.DueAt.Equal(now.AddDate(0, 0, 1)) {
		t.Fatalf("again due %v", s.DueAt)
	}
	if s.Easiness >= beforeEF {
		t.Fatalf("EF should drop on again: %v -> %v", beforeEF, s.Easiness)
	}
}

func TestMinEasiness(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewState(now)
	s.Easiness = 1.3
	s = Schedule(s, RatingAgain, now)
	if s.Easiness != 1.3 {
		t.Fatalf("EF %v", s.Easiness)
	}
}

func TestHardIsPass(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewState(now)
	s = Schedule(s, RatingHard, now)
	if s.Repetitions != 1 || s.IntervalDays != 1 {
		t.Fatalf("hard: %+v", s)
	}
	if s.Easiness >= 2.5 {
		t.Fatalf("EF should drop on hard: %v", s.Easiness)
	}
}

func TestEasyRaisesEF(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := Schedule(NewState(now), RatingEasy, now)
	if s.Easiness <= 2.5 || s.IntervalDays != 1 || s.Repetitions != 1 {
		t.Fatalf("%+v", s)
	}
}
