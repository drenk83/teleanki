package domain

import (
	"testing"
	"time"
)

func TestParseMode(t *testing.T) {
	t.Parallel()
	for _, m := range []Mode{ModeRecall, ModeQuiz, ModeTypein} {
		got, err := ParseMode(string(m))
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", m, err)
		}
		if got != m {
			t.Fatalf("ParseMode(%q) = %q", m, got)
		}
	}
	if _, err := ParseMode("fsrs"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if Mode("").Valid() {
		t.Fatal("empty mode must be invalid")
	}
}

func TestNormalizeDeckName(t *testing.T) {
	t.Parallel()
	got, err := NormalizeDeckName("  Колода  ")
	if err != nil || got != "Колода" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeDeckName("   "); err == nil {
		t.Fatal("expected error for empty name")
	}
	long := stringsRepeat("я", MaxDeckNameRunes+1)
	if _, err := NormalizeDeckName(long); err == nil {
		t.Fatal("expected error for long name")
	}
}

func TestNormalizeDailyLimit(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeDailyLimit(0); err == nil {
		t.Fatal("expected error")
	}
	got, err := NormalizeDailyLimit(37)
	if err != nil || got != 37 {
		t.Fatalf("got %d %v", got, err)
	}
	if _, err := NormalizeDailyLimit(201); err == nil {
		t.Fatal("expected error")
	}
}

func TestRemainingToday(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 25, 15, 0, 0, 0, time.UTC)
	u := User{DailyLimit: 20, ReviewsToday: 7, ReviewsOnDate: now}
	if u.RemainingToday(now) != 13 {
		t.Fatalf("got %d", u.RemainingToday(now))
	}
	yesterday := now.AddDate(0, 0, -1)
	u.ReviewsOnDate = yesterday
	if u.RemainingToday(now) != 20 {
		t.Fatalf("new day: got %d", u.RemainingToday(now))
	}
}

func TestBuildQuizChoices(t *testing.T) {
	t.Parallel()
	got, err := BuildQuizChoices("да", []string{" нет ", "да", "", "может"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "да" {
		t.Fatalf("got %#v", got)
	}
	if _, err := BuildQuizChoices("да", []string{"1", "2", "3", "4", "5", "6"}); err == nil {
		t.Fatal("expected error for too many choices")
	}
	if _, err := BuildQuizChoices("да", nil); err == nil {
		t.Fatal("expected error for too few choices")
	}
	if err := ValidateQuizChoices("да", []string{"нет", "может"}); err == nil {
		t.Fatal("expected error when back missing")
	}
	if err := ValidateQuizChoices("да", []string{"да", "да"}); err == nil {
		t.Fatal("expected error for duplicates")
	}
}

func TestMatchTypein(t *testing.T) {
	t.Parallel()
	if !MatchTypein("  Go ", "go") {
		t.Fatal("expected match")
	}
	if MatchTypein("golang", "go") {
		t.Fatal("expected mismatch")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]rune, 0, n)
	r := []rune(s)[0]
	for i := 0; i < n; i++ {
		out = append(out, r)
	}
	return string(out)
}
