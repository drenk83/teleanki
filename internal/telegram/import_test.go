package telegram

import (
	"testing"

	"github.com/drenk83/teleanki/internal/config"
)

func TestParseImportEmptyCards(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{"deck":"Колода","cards":[]}`))
	if err == nil || err.Error() != config.ImportEmptyCards {
		t.Fatalf("got %v", err)
	}
}

func TestParseImportBadJSON(t *testing.T) {
	t.Parallel()
	_, err := parseImport([]byte(`{`))
	if err == nil || err.Error() != config.ImportBadJSON {
		t.Fatalf("got %v", err)
	}
}

func TestParseImportQuizFoldBack(t *testing.T) {
	t.Parallel()
	got, err := parseImport([]byte(`{"deck":"Колода","cards":[{"front":"q","back":"верный","mode":"quiz","choices":["Верный","нет"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Cards[0].Back != "верный" || len(got.Cards[0].Choices) != 2 {
		t.Fatalf("%#v", got.Cards[0])
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
