package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type importFile struct {
	Deck  string       `json:"deck"`
	Cards []importCard `json:"cards"`
}

type importCard struct {
	Front      string   `json:"front"`
	Back       string   `json:"back"`
	Mode       string   `json:"mode"`
	Choices    []string `json:"choices"`
	Reversible bool     `json:"reversible"`
}

var (
	errImportTooLarge       = errors.New(config.ImportTooLarge)
	errImportBadJSON        = errors.New(config.ImportBadJSON)
	errImportBadDeckName    = errors.New(config.ImportBadDeckName)
	errImportBadDefaultMode = errors.New(config.ImportBadDefaultMode)
	errImportEmptyCards     = errors.New(config.ImportEmptyCards)
	errImportTooManyCards   = errors.New(config.ImportTooManyCards)
)

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
	if err := h.cards.CreateMany(ctx, userID, sess.Import.ExistingID, sess.Import.Cards); err != nil {
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
		name, err := domain.ConflictDeckName(original, i)
		if err != nil {
			return "", err
		}
		_, err = h.decks.GetByUserAndName(ctx, userID, name)
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
	return downloadTelegramFile(ctx, b, fileID, maxImportB)
}

func parseImport(data []byte) (*importDraft, error) {
	if len(data) > maxImportB {
		return nil, errImportTooLarge
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, errImportBadJSON
	}
	if _, ok := probe["default_mode"]; ok {
		return nil, errImportBadDefaultMode
	}
	var raw importFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errImportBadJSON
	}
	name, err := domain.NormalizeDeckName(raw.Deck)
	if err != nil {
		return nil, errImportBadDeckName
	}
	fillMode := domain.ModeRecall
	if len(raw.Cards) == 0 {
		return nil, errImportEmptyCards
	}
	if len(raw.Cards) > maxImportN {
		return nil, errImportTooManyCards
	}
	cards := make([]domain.Card, 0, len(raw.Cards))
	for i, rc := range raw.Cards {
		cardMode := fillMode
		if strings.TrimSpace(rc.Mode) != "" {
			m, err := domain.ParseMode(rc.Mode)
			if err != nil {
				return nil, fmt.Errorf(config.ImportBadCardMode, i+1)
			}
			cardMode = m
		}
		c, err := domain.NewCardWithChoices(0, rc.Front, rc.Back, cardMode, rc.Choices, rc.Reversible)
		if err != nil {
			if _, nerr := domain.NormalizeCardText(rc.Front); nerr != nil {
				return nil, fmt.Errorf(config.ImportBadCardFront, i+1)
			}
			if _, nerr := domain.NormalizeCardText(rc.Back); nerr != nil {
				return nil, fmt.Errorf(config.ImportBadCardBack, i+1)
			}
			return nil, fmt.Errorf(config.ImportBadCardChoices, i+1)
		}
		cards = append(cards, c)
	}
	return &importDraft{Name: name, Cards: cards}, nil
}

func isJSONDoc(doc *models.Document) bool {
	if strings.EqualFold(doc.MimeType, "application/json") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(doc.FileName), ".json")
}
