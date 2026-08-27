package telegram

import (
	"errors"
	"testing"
)

func TestParseImportEmptyCards(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{"deck":"Колода","cards":[]}`))
	if !errors.Is(err, errImportEmptyCards) {
		t.Fatalf("got %v", err)
	}
}

func TestParseImportBadJSON(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{`))
	if !errors.Is(err, errImportBadJSON) {
		t.Fatalf("got %v", err)
	}
}

func TestParseImportQuizBackInChoices(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{"deck":"Колода","cards":[{"front":"q","back":"верный","mode":"quiz","choices":["Верный","нет"]}]}`))
	if err == nil {
		t.Fatal("expected error when back is in choices")
	}
}

func TestParseImportQuizDistractors(t *testing.T) {
	t.Parallel()
	got, err := parseImport([]byte(`{"deck":"Колода","cards":[{"front":"q","back":"верный","mode":"quiz","choices":["нет","может"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Cards[0].Back != "верный" || len(got.Cards[0].Choices) != 2 || got.Cards[0].Choices[0] != "нет" {
		t.Fatalf("%#v", got.Cards[0])
	}
}

func TestParseImportRejectsDefaultMode(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{"deck":"Колода","default_mode":"recall","cards":[{"front":"q","back":"a"}]}`))
	if !errors.Is(err, errImportBadDefaultMode) {
		t.Fatalf("got %v", err)
	}
}

func TestParseImportOK(t *testing.T) {
	t.Parallel()
	got, err := parseImport([]byte(`{"deck":"Колода","cards":[{"front":"q","back":"a"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Колода" || len(got.Cards) != 1 {
		t.Fatalf("%#v", got)
	}
}
