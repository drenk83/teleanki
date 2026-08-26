package telegram

import (
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/go-telegram/bot/models"
)

var (
	htmlTagDetectRe = regexp.MustCompile(`(?i)</?(b|strong|i|em|u|s|strike|del|code|pre|blockquote|tg-spoiler|tg-emoji|a)\b`)
	mdFenceRe       = regexp.MustCompile("(?s)```([A-Za-z0-9_+-]*)\n(.*?)```")
	mdCodeRe        = regexp.MustCompile("`([^`]+)`")
	mdBoldRe        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdStrikeRe      = regexp.MustCompile(`~~(.+?)~~`)
	mdItalicRe      = regexp.MustCompile(`\*(.+?)\*`)
)

func formattedCardText(msg *models.Message) string {
	if msg == nil {
		return ""
	}
	if !hasFormatEntities(msg.Entities) {
		return msg.Text
	}
	return entitiesToHTML(msg.Text, msg.Entities)
}

func hasFormatEntities(ents []models.MessageEntity) bool {
	for _, e := range ents {
		switch e.Type {
		case models.MessageEntityTypeBold,
			models.MessageEntityTypeItalic,
			models.MessageEntityTypeUnderline,
			models.MessageEntityTypeStrikethrough,
			models.MessageEntityTypeSpoiler,
			models.MessageEntityTypeBlockquote,
			models.MessageEntityTypeExpandableBlockquote,
			models.MessageEntityTypeCode,
			models.MessageEntityTypePre,
			models.MessageEntityTypeTextLink,
			models.MessageEntityTypeTextMention,
			models.MessageEntityTypeCustomEmoji:
			return true
		}
	}
	return false
}

func messageHTML(s string) string {
	if htmlTagDetectRe.MatchString(s) {
		return s
	}
	return markdownToHTML(html.EscapeString(s))
}

func markdownToHTML(s string) string {
	s = mdFenceRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdFenceRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		lang, code := sub[1], sub[2]
		if lang != "" {
			return `<pre><code class="language-` + lang + `">` + code + `</code></pre>`
		}
		return `<pre>` + code + `</pre>`
	})
	s = mdCodeRe.ReplaceAllString(s, `<code>$1</code>`)
	s = mdBoldRe.ReplaceAllString(s, `<b>$1</b>`)
	s = mdStrikeRe.ReplaceAllString(s, `<s>$1</s>`)
	s = mdItalicRe.ReplaceAllString(s, `<i>$1</i>`)
	return quoteLines(s)
}

func quoteLines(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	in := false
	for i, line := range lines {
		q, ok := strings.CutPrefix(line, "&gt;")
		if ok {
			q = strings.TrimPrefix(q, " ")
			if !in {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString("<blockquote>")
				in = true
			} else {
				b.WriteByte('\n')
			}
			b.WriteString(q)
			continue
		}
		if in {
			b.WriteString("</blockquote>")
			in = false
		}
		if i > 0 && b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	if in {
		b.WriteString("</blockquote>")
	}
	return b.String()
}

type htmlMark struct {
	pos  int
	pri  int
	open bool
	tag  string
}

func entitiesToHTML(text string, entities []models.MessageEntity) string {
	u16 := utf16.Encode([]rune(text))
	var marks []htmlMark
	for i, e := range entities {
		open, close := entityTags(e)
		if open == "" {
			continue
		}
		start, end := e.Offset, e.Offset+e.Length
		if start < 0 || end > len(u16) || start >= end {
			continue
		}
		marks = append(marks,
			htmlMark{pos: start, pri: i, open: true, tag: open},
			htmlMark{pos: end, pri: i, open: false, tag: close},
		)
	}
	sort.SliceStable(marks, func(i, j int) bool {
		if marks[i].pos != marks[j].pos {
			return marks[i].pos < marks[j].pos
		}
		if marks[i].open != marks[j].open {
			return !marks[i].open
		}
		if marks[i].open {
			return marks[i].pri < marks[j].pri
		}
		return marks[i].pri > marks[j].pri
	})
	var b strings.Builder
	prev := 0
	for _, m := range marks {
		if m.pos > prev {
			b.WriteString(html.EscapeString(string(utf16.Decode(u16[prev:m.pos]))))
		}
		b.WriteString(m.tag)
		prev = m.pos
	}
	if prev < len(u16) {
		b.WriteString(html.EscapeString(string(utf16.Decode(u16[prev:]))))
	}
	return b.String()
}

func entityTags(e models.MessageEntity) (open, close string) {
	switch e.Type {
	case models.MessageEntityTypeBold:
		return "<b>", "</b>"
	case models.MessageEntityTypeItalic:
		return "<i>", "</i>"
	case models.MessageEntityTypeUnderline:
		return "<u>", "</u>"
	case models.MessageEntityTypeStrikethrough:
		return "<s>", "</s>"
	case models.MessageEntityTypeSpoiler:
		return "<tg-spoiler>", "</tg-spoiler>"
	case models.MessageEntityTypeBlockquote:
		return "<blockquote>", "</blockquote>"
	case models.MessageEntityTypeExpandableBlockquote:
		return "<blockquote expandable>", "</blockquote>"
	case models.MessageEntityTypeCode:
		return "<code>", "</code>"
	case models.MessageEntityTypePre:
		if e.Language != "" {
			return `<pre><code class="language-` + html.EscapeString(e.Language) + `">`, "</code></pre>"
		}
		return "<pre>", "</pre>"
	case models.MessageEntityTypeTextLink:
		return `<a href="` + html.EscapeString(e.URL) + `">`, "</a>"
	case models.MessageEntityTypeTextMention:
		if e.User != nil {
			return `<a href="tg://user?id=` + strconv.FormatInt(e.User.ID, 10) + `">`, "</a>"
		}
	case models.MessageEntityTypeCustomEmoji:
		return `<tg-emoji emoji-id="` + html.EscapeString(e.CustomEmojiID) + `">`, "</tg-emoji>"
	}
	return "", ""
}
