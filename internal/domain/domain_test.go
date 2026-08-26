package domain

import (
	"testing"
	"time"
	"unicode/utf8"
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

func TestShareCode(t *testing.T) {
	t.Parallel()
	code, err := NewShareCode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeShareCode("  " + code + "  ")
	if err != nil || got != code {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeShareCode("short"); err == nil {
		t.Fatal("expected error")
	}
}

func TestListTitle(t *testing.T) {
	t.Parallel()
	d := Deck{UserID: 1, Name: "EN", OwnerUsername: "ann"}
	if d.ListTitle(1) != "EN" {
		t.Fatalf("own: %q", d.ListTitle(1))
	}
	if d.ListTitle(2) != "EN · @ann" {
		t.Fatalf("shared: %q", d.ListTitle(2))
	}
}

func TestNormalizeNotifyHour(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeNotifyHour(-1); err == nil {
		t.Fatal("expected error")
	}
	got, err := NormalizeNotifyHour(19)
	if err != nil || got != 19 {
		t.Fatalf("%d %v", got, err)
	}
}

func TestImageExt(t *testing.T) {
	t.Parallel()
	ext, err := ImageExt("image/png")
	if err != nil || ext != ".png" {
		t.Fatalf("%q %v", ext, err)
	}
	if _, err := ImageExt("image/gif"); err == nil {
		t.Fatal("expected error")
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

func TestConflictDeckName(t *testing.T) {
	t.Parallel()
	got, err := ConflictDeckName("Колода", 2)
	if err != nil || got != "Колода (2)" {
		t.Fatalf("got %q %v", got, err)
	}
	long := stringsRepeat("я", MaxDeckNameRunes)
	got, err = ConflictDeckName(long, 2)
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(got) > MaxDeckNameRunes {
		t.Fatalf("too long: %d %q", utf8.RuneCountInString(got), got)
	}
	if _, err := NormalizeDeckName(got); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeCardText(t *testing.T) {
	t.Parallel()
	got, err := NormalizeCardText("  hi  ")
	if err != nil || got != "hi" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeCardText("   "); err == nil {
		t.Fatal("expected error for empty")
	}
	if _, err := NormalizeCardText(stringsRepeat("я", MaxCardTextRunes+1)); err == nil {
		t.Fatal("expected error for long")
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
	u := User{DailyLimit: 20, ReviewsToday: 7, ReviewsOnDate: DayDate(now)}
	if u.RemainingToday(now) != 13 {
		t.Fatalf("got %d", u.RemainingToday(now))
	}
	u.ReviewsOnDate = DayDate(now.AddDate(0, 0, -1))
	if u.RemainingToday(now) != 20 {
		t.Fatalf("new day: got %d", u.RemainingToday(now))
	}
}

func TestRemainingTodayMoscowMidnight(t *testing.T) {
	t.Parallel()
	before := time.Date(2026, 8, 25, 20, 59, 0, 0, time.UTC)
	after := time.Date(2026, 8, 25, 21, 0, 0, 0, time.UTC)
	u := User{DailyLimit: 20, ReviewsToday: 7, ReviewsOnDate: DayDate(before)}
	if u.RemainingToday(before) != 13 {
		t.Fatalf("before midnight MSK: %d", u.RemainingToday(before))
	}
	if u.RemainingToday(after) != 20 {
		t.Fatalf("after midnight MSK: %d", u.RemainingToday(after))
	}
}

func TestDayDateIsMoscowCivilDate(t *testing.T) {
	t.Parallel()
	got := DayDate(time.Date(2026, 8, 25, 21, 30, 0, 0, time.UTC))
	want := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildQuizChoices(t *testing.T) {
	t.Parallel()
	got, err := BuildQuizChoices("да", []string{" нет ", "да", "", "может"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "нет" || got[1] != "может" {
		t.Fatalf("got %#v", got)
	}
	if _, err := BuildQuizChoices("да", []string{"1", "2", "3", "4", "5", "6"}); err == nil {
		t.Fatal("expected error for too many distractors")
	}
	if _, err := BuildQuizChoices("да", nil); err == nil {
		t.Fatal("expected error for too few distractors")
	}
	if err := ValidateQuizChoices("да", []string{"нет", "может"}); err != nil {
		t.Fatalf("valid distractors: %v", err)
	}
	if err := ValidateQuizChoices("да", []string{"нет", "Нет"}); err == nil {
		t.Fatal("expected error for fold duplicates")
	}
	if err := ValidateQuizChoices("да", []string{"Да", "нет"}); err == nil {
		t.Fatal("expected error when distractor is back")
	}
	got, err = BuildQuizChoices("да", []string{"ДА", "нет"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "нет" {
		t.Fatalf("fold skip back: %#v", got)
	}
}

func TestQuizButtons(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "да", ModeQuiz, []string{"нет", "может"}, false)
	if err != nil {
		t.Fatal(err)
	}
	got := c.QuizButtons()
	if len(got) != 3 || got[0] != "да" || got[1] != "нет" || got[2] != "может" {
		t.Fatalf("%#v", got)
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
	if !MatchTypein("**Go**", "<b>go</b>") {
		t.Fatal("expected match ignoring markup")
	}
	if !MatchTypein("  `Hi` ", "hi") {
		t.Fatal("expected match stripped code")
	}
}

func TestCanFlip(t *testing.T) {
	t.Parallel()
	recall := Card{Mode: ModeRecall, Reversible: true}
	if !recall.CanFlip() {
		t.Fatal("recall reversible must flip")
	}
	typein := Card{Mode: ModeTypein, Reversible: true}
	if !typein.CanFlip() {
		t.Fatal("typein reversible must flip")
	}
	if (Card{Mode: ModeQuiz, Reversible: true}).CanFlip() {
		t.Fatal("quiz must not flip")
	}
	if (Card{Mode: ModeRecall, Reversible: false}).CanFlip() {
		t.Fatal("non-reversible must not flip")
	}
}

func TestPromptAnswer(t *testing.T) {
	t.Parallel()
	c := Card{Front: "hello", Back: "привет", Mode: ModeRecall, Reversible: true}
	p, a := c.PromptAnswer(false)
	if p != "hello" || a != "привет" {
		t.Fatalf("normal: %q %q", p, a)
	}
	p, a = c.PromptAnswer(true)
	if p != "привет" || a != "hello" {
		t.Fatalf("flipped: %q %q", p, a)
	}
	quiz := Card{Front: "q", Back: "a", Mode: ModeQuiz, Reversible: true}
	p, a = quiz.PromptAnswer(true)
	if p != "q" || a != "a" {
		t.Fatalf("quiz ignore flip: %q %q", p, a)
	}
	_, want := c.PromptAnswer(true)
	if !MatchTypein("  Hello ", want) {
		t.Fatal("typein should match hidden side")
	}
}

func TestNewCardQuizNotReversible(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "a", ModeQuiz, []string{"b"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Reversible {
		t.Fatal("quiz must not be reversible")
	}
	if err := ValidateQuizChoices(c.Back, c.Choices); err != nil {
		t.Fatal(err)
	}
}

func TestWithModeQuizWithoutChoices(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "a", ModeRecall, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.WithMode(ModeQuiz); err != ErrQuizNeedsChoices {
		t.Fatalf("got %v", err)
	}
	if c.Mode != ModeRecall {
		t.Fatal("original must stay recall")
	}
}

func TestWithModeAwayFromQuizClearsChoices(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "a", ModeQuiz, []string{"b"}, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.WithMode(ModeRecall)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != ModeRecall || len(got.Choices) != 0 {
		t.Fatalf("mode=%s choices=%v", got.Mode, got.Choices)
	}
}

func TestBecomeQuizWithBack(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "Париж", ModeQuiz, []string{"Лион", "Берлин"}, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.BecomeQuizWithBack("Лион", []string{"Мадрид", "Рим"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Back != "Лион" || got.Mode != ModeQuiz {
		t.Fatalf("back=%q mode=%s", got.Back, got.Mode)
	}
	if err := ValidateQuizChoices(got.Back, got.Choices); err != nil {
		t.Fatal(err)
	}
	if _, err := c.BecomeQuizWithBack("   ", []string{"x"}); err == nil {
		t.Fatal("expected error for empty back")
	}
}

func TestBecomeQuiz(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "a", ModeRecall, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.BecomeQuiz([]string{"b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != ModeQuiz || got.Reversible {
		t.Fatalf("got mode=%s rev=%v", got.Mode, got.Reversible)
	}
	if err := ValidateQuizChoices(got.Back, got.Choices); err != nil {
		t.Fatal(err)
	}
}

func TestWithBackKeepsDistractors(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "Париж", ModeQuiz, []string{"Лондон", "Берлин"}, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.WithBack("Лион")
	if err != nil {
		t.Fatal(err)
	}
	if got.Back != "Лион" {
		t.Fatalf("back=%q", got.Back)
	}
	if err := ValidateQuizChoices(got.Back, got.Choices); err != nil {
		t.Fatal(err)
	}
	if len(got.Choices) != 2 || got.Choices[0] != "Лондон" || got.Choices[1] != "Берлин" {
		t.Fatalf("choices=%v", got.Choices)
	}
}

func TestWithBackQuizDuplicateFails(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "Париж", ModeQuiz, []string{"Лион", "Берлин"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.WithBack("Лион"); err == nil {
		t.Fatal("expected error when new back duplicates a choice")
	}
	if _, err := c.WithBack("ЛИОН"); err == nil {
		t.Fatal("expected error when new back fold-duplicates a choice")
	}
}

func TestToggleReverseQuizStaysOff(t *testing.T) {
	t.Parallel()
	c, err := NewCard(1, "q", "a", ModeQuiz, []string{"b"}, false)
	if err != nil {
		t.Fatal(err)
	}
	got := c.ToggleReverse()
	if got.Reversible {
		t.Fatal("quiz toggle must stay off")
	}
}

func TestNewCardWithChoicesImport(t *testing.T) {
	t.Parallel()
	c, err := NewCardWithChoices(1, "q", "верный", ModeQuiz, []string{"нет", "может"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Reversible {
		t.Fatal("quiz import must strip reversible")
	}
	if len(c.Choices) != 2 || c.Choices[0] != "нет" {
		t.Fatalf("distractors=%v", c.Choices)
	}
	if _, err := NewCardWithChoices(1, "q", "верный", ModeQuiz, []string{"верный", "нет"}, false); err == nil {
		t.Fatal("expected error when back is in choices")
	}
	if _, err := NewCardWithChoices(1, "q", "верный", ModeQuiz, []string{"Верный", "нет"}, false); err == nil {
		t.Fatal("expected error when fold back is in choices")
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
