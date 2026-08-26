package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type importFile struct {
	Deck        string       `json:"deck"`
	DefaultMode string       `json:"default_mode"`
	Cards       []importCard `json:"cards"`
}

type importCard struct {
	Front   string   `json:"front"`
	Back    string   `json:"back"`
	Mode    string   `json:"mode"`
	Choices []string `json:"choices"`
}

func (h *Bot) documentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil || update.Message.Document == nil {
		return
	}
	doc := update.Message.Document
	chatID := update.Message.Chat.ID
	tgID := update.Message.From.ID
	if !isJSONDoc(doc) || doc.FileSize > maxImportB {
		h.send(ctx, b, chatID, config.ImportBadFile, nil)
		return
	}
	user, err := h.currentUser(ctx, tgID, update.Message.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	data, err := h.downloadFile(ctx, b, doc.FileID)
	if err != nil {
		slog.Error("Failed to download file", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	draft, err := parseImport(data)
	if err != nil {
		h.send(ctx, b, chatID, fmt.Sprintf(config.ImportBadData, err.Error()), nil)
		return
	}
	existing, err := h.decks.GetByUserAndName(ctx, user.ID, draft.Name)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		slog.Error("Failed to get deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	if errors.Is(err, storage.ErrNotFound) {
		h.finishImportNew(ctx, b, tgID, chatID, user.ID, draft)
		return
	}
	draft.ExistingID = existing.ID
	h.sessions.set(tgID, &session{State: stateImportConflict, Import: draft})
	h.send(ctx, b, chatID, fmt.Sprintf(config.ImportConflict, draft.Name), kb(
		row(btn(config.BtnImportNew, "i:new")),
		row(btn(config.BtnImportAppend, "i:append")),
		row(btn(config.BtnImportCancel, "i:cancel")),
	))
}

func (h *Bot) finishImportNew(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64, draft *importDraft) {
	name := draft.Name
	if draft.ExistingID != 0 {
		n, err := h.nextFreeDeckName(ctx, userID, draft.Name)
		if err != nil {
			slog.Error("Failed to pick deck name", "error", err)
			h.send(ctx, b, chatID, config.TryAgain, nil)
			return
		}
		name = n
	}
	d, err := h.decks.CreateWithCards(ctx, userID, name, draft.Cards)
	if err != nil {
		slog.Error("Failed to import deck", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	h.sessions.clear(tgID)
	h.showDeck(ctx, b, chatID, userID, d.ID)
}

func (h *Bot) finishImportAppend(ctx context.Context, b *bot.Bot, tgID, chatID, userID int64) {
	sess := h.sessions.get(tgID)
	if sess.Import == nil || sess.Import.ExistingID == 0 {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	if _, err := h.deckOf(ctx, userID, sess.Import.ExistingID); err != nil {
		h.sessions.clear(tgID)
		h.send(ctx, b, chatID, config.SessionExpired, nil)
		return
	}
	for i, c := range sess.Import.Cards {
		if c.Mode != domain.ModeQuiz {
			continue
		}
		if err := domain.ValidateQuizChoices(c.Back, c.Choices); err != nil {
			h.send(ctx, b, chatID, fmt.Sprintf(config.ImportBadData, fmt.Sprintf("карточка %d: некорректные варианты", i+1)), nil)
			return
		}
	}
	if err := h.cards.CreateMany(ctx, sess.Import.ExistingID, sess.Import.Cards); err != nil {
		slog.Error("Failed to append cards", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	deckID := sess.Import.ExistingID
	h.sessions.clear(tgID)
	h.showDeck(ctx, b, chatID, userID, deckID)
}

func (h *Bot) nextFreeDeckName(ctx context.Context, userID int64, original string) (string, error) {
	for i := 2; i < 100; i++ {
		name := fmt.Sprintf("%s (%d)", original, i)
		_, err := h.decks.GetByUserAndName(ctx, userID, name)
		if errors.Is(err, storage.ErrNotFound) {
			return name, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("no free deck name")
}

func (h *Bot) downloadFile(ctx context.Context, b *bot.Bot, fileID string) ([]byte, error) {
	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, err
	}
	link := b.FileDownloadLink(f)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxImportB+1))
}

func parseImport(data []byte) (*importDraft, error) {
	if len(data) > maxImportB {
		return nil, errors.New("файл слишком большой")
	}
	var raw importFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.New("это не JSON")
	}
	name, err := domain.NormalizeDeckName(raw.Deck)
	if err != nil {
		return nil, errors.New("некорректное имя колоды")
	}
	fillMode := domain.ModeRecall
	if strings.TrimSpace(raw.DefaultMode) != "" {
		m, err := domain.ParseMode(raw.DefaultMode)
		if err != nil {
			return nil, errors.New("некорректный default_mode")
		}
		fillMode = m
	}
	if len(raw.Cards) > maxImportN {
		return nil, errors.New("слишком много карточек")
	}
	cards := make([]domain.Card, 0, len(raw.Cards))
	for i, rc := range raw.Cards {
		front, err := domain.NormalizeCardText(rc.Front)
		if err != nil {
			return nil, fmt.Errorf("карточка %d: некорректный вопрос", i+1)
		}
		back, err := domain.NormalizeCardText(rc.Back)
		if err != nil {
			return nil, fmt.Errorf("карточка %d: некорректный ответ", i+1)
		}
		cardMode := fillMode
		if strings.TrimSpace(rc.Mode) != "" {
			m, err := domain.ParseMode(rc.Mode)
			if err != nil {
				return nil, fmt.Errorf("карточка %d: некорректный режим", i+1)
			}
			cardMode = m
		}
		c := domain.Card{Front: front, Back: back, Mode: cardMode, Choices: []string{}}
		if cardMode == domain.ModeQuiz {
			choices := trimChoices(rc.Choices)
			if err := domain.ValidateQuizChoices(back, choices); err != nil {
				return nil, fmt.Errorf("карточка %d: некорректные варианты", i+1)
			}
			c.Choices = choices
		}
		cards = append(cards, c)
	}
	return &importDraft{Name: name, Cards: cards}, nil
}

func trimChoices(choices []string) []string {
	out := make([]string, 0, len(choices))
	for _, c := range choices {
		out = append(out, strings.TrimSpace(c))
	}
	return out
}

func isJSONDoc(doc *models.Document) bool {
	if strings.EqualFold(doc.MimeType, "application/json") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(doc.FileName), ".json")
}
