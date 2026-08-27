package telegram

import (
	"testing"

	"github.com/drenk83/teleanki/internal/domain"
)

func TestCardDraftSetFront(t *testing.T) {
	t.Parallel()
	_, err := cardDraft{}.setFront("  ", "")
	if err == nil {
		t.Fatal("empty")
	}
	d, err := cardDraft{}.setFront("  Q  ", "f.jpg")
	if err != nil || d.Front != "Q" || d.FrontImage != "f.jpg" {
		t.Fatalf("%#v %v", d, err)
	}
}

func TestCardDraftSetBack(t *testing.T) {
	t.Parallel()
	_, err := cardDraft{Front: "Q"}.setBack("", "b.jpg")
	if err == nil {
		t.Fatal("empty")
	}
	d, err := cardDraft{Front: "Q"}.setBack("A", "b.jpg")
	if err != nil || d.Back != "A" || d.BackImage != "b.jpg" {
		t.Fatalf("%#v %v", d, err)
	}
}

func TestCardDraftAfterBack(t *testing.T) {
	t.Parallel()
	d, err := cardDraft{Front: "Q"}.setBack("A", "")
	if err != nil {
		t.Fatal(err)
	}
	d, st := d.afterBack()
	if st != stateCardMode || d.Mode != "" {
		t.Fatalf("%#v %s", d, st)
	}
	d, err = cardDraft{Front: "Q"}.setBack("A", "b.jpg")
	if err != nil {
		t.Fatal(err)
	}
	d, st = d.afterBack()
	if st != stateCardReverse || d.Mode != domain.ModeRecall || d.Reversible || d.BackImage != "b.jpg" {
		t.Fatalf("%#v %s", d, st)
	}
}

func TestCardDraftSetMode(t *testing.T) {
	t.Parallel()
	d, st := cardDraft{Reversible: true}.setMode(domain.ModeQuiz)
	if st != stateCardChoices || d.Mode != domain.ModeQuiz || d.Reversible {
		t.Fatalf("%#v %s", d, st)
	}
	d, st = cardDraft{}.setMode(domain.ModeRecall)
	if st != stateCardReverse || d.Mode != domain.ModeRecall {
		t.Fatalf("%#v %s", d, st)
	}
	d, st = cardDraft{}.setMode(domain.ModeTypein)
	if st != stateCardReverse || d.Mode != domain.ModeTypein {
		t.Fatalf("%#v %s", d, st)
	}
}

func TestCardDraftCommitNew(t *testing.T) {
	t.Parallel()
	d := cardDraft{Front: "Q", Back: "A", Mode: domain.ModeRecall, Reversible: true, FrontImage: "f.jpg", BackImage: "b.jpg"}
	c, err := d.commitNew(9, nil)
	if err != nil || c.DeckID != 9 || c.Front != "Q" || c.FrontImage != "f.jpg" || c.BackImage != "b.jpg" || !c.Reversible {
		t.Fatalf("%#v %v", c, err)
	}
	d.Mode = domain.ModeQuiz
	if _, err := d.commitNew(9, []string{"A", "A"}); err == nil {
		t.Fatal("expected invalid choices")
	}
	c, err = d.commitNew(9, []string{"no", "maybe"})
	if err != nil || len(c.Choices) != 2 || c.Reversible {
		t.Fatalf("%#v %v", c, err)
	}
}

func TestApplyFrontEdit(t *testing.T) {
	t.Parallel()
	c := domain.Card{ID: 1, Front: "old", Back: "A", FrontImage: "old.jpg"}
	next, err := applyFrontEdit(c, "new", "new.jpg")
	if err != nil || next.Front != "new" || next.FrontImage != "new.jpg" {
		t.Fatalf("%#v %v", next, err)
	}
	if _, err := applyFrontEdit(c, "  ", ""); err == nil {
		t.Fatal("empty")
	}
}

func TestApplyBackEdit(t *testing.T) {
	t.Parallel()
	c := domain.Card{ID: 1, Front: "Q", Back: "yes", Mode: domain.ModeQuiz, Choices: []string{"no"}, BackImage: "old.jpg"}
	got, err := applyBackEdit(c, "no", "new.jpg")
	if err != nil || !got.NeedChoices || got.DraftBack != "no" || got.Card.ID != 0 {
		t.Fatalf("%#v %v", got, err)
	}
	ok, err := applyBackEdit(c, "yep", "new.jpg")
	if err != nil || ok.NeedChoices || ok.Card.Back != "yep" || ok.Card.BackImage != "new.jpg" {
		t.Fatalf("%#v %v", ok, err)
	}
	if _, err := applyBackEdit(c, "  ", ""); err == nil {
		t.Fatal("empty")
	}
}
