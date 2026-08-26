package telegram

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestFormattedCardTextPlain(t *testing.T) {
	t.Parallel()
	msg := &models.Message{Text: "**keep**"}
	if got := formattedCardText(msg); got != "**keep**" {
		t.Fatalf("got %q", got)
	}
}

func TestEntitiesToHTML(t *testing.T) {
	t.Parallel()
	text := "hello"
	got := entitiesToHTML(text, []models.MessageEntity{
		{Type: models.MessageEntityTypeBold, Offset: 0, Length: 5},
	})
	if got != "<b>hello</b>" {
		t.Fatalf("got %q", got)
	}
}

func TestEntitiesToHTMLUTF16(t *testing.T) {
	t.Parallel()
	text := "🙂ok"
	got := entitiesToHTML(text, []models.MessageEntity{
		{Type: models.MessageEntityTypeItalic, Offset: 2, Length: 2},
	})
	if got != "🙂<i>ok</i>" {
		t.Fatalf("got %q", got)
	}
}

func TestMessageHTMLMarkdown(t *testing.T) {
	t.Parallel()
	got := messageHTML("**bold** *it* ~~x~~ `c`")
	if got != "<b>bold</b> <i>it</i> <s>x</s> <code>c</code>" {
		t.Fatalf("got %q", got)
	}
}

func TestMessageHTMLKeepsTags(t *testing.T) {
	t.Parallel()
	in := "Q: <b>hi</b>"
	if got := messageHTML(in); got != in {
		t.Fatalf("got %q", got)
	}
}

func TestMessageHTMLFenceAndQuote(t *testing.T) {
	t.Parallel()
	got := messageHTML("```go\ncode\n```\n> said")
	if got != "<pre><code class=\"language-go\">code\n</code></pre>\n<blockquote>said</blockquote>" {
		t.Fatalf("got %q", got)
	}
}
