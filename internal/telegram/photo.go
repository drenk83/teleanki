package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Bot) photoHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil || len(update.Message.Photo) == 0 {
		return
	}
	if update.Message.Chat.Type != models.ChatTypePrivate {
		return
	}
	tgID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	user, err := h.currentUser(ctx, tgID, update.Message.From.Username)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		h.send(ctx, b, chatID, config.TryAgain, nil)
		return
	}
	caption := formattedCaption(update.Message)
	sess := h.sessions.get(tgID)
	switch sess.State {
	case stateCardFront, stateCardBack, stateEditFront, stateEditBack:
	default:
		h.send(ctx, b, chatID, config.UnknownCommand, nil)
		return
	}
	if caption == "" {
		h.send(ctx, b, chatID, config.PhotoNeedCaption, nil)
		return
	}
	name, err := h.saveMessagePhoto(ctx, b, update.Message)
	if err != nil {
		slog.Error("Failed to save photo", "error", err)
		h.send(ctx, b, chatID, config.PhotoBadFile, nil)
		return
	}
	switch sess.State {
	case stateCardFront:
		h.addCardFrontPhoto(ctx, b, tgID, chatID, caption, name)
	case stateCardBack:
		h.addCardBackPhoto(ctx, b, tgID, chatID, caption, name)
	case stateEditFront:
		h.editCardFrontPhoto(ctx, b, tgID, chatID, user.ID, caption, name)
	case stateEditBack:
		h.editCardBackPhoto(ctx, b, tgID, chatID, user.ID, caption, name)
	}
}

func formattedCaption(msg *models.Message) string {
	if !hasFormatEntities(msg.CaptionEntities) {
		return msg.Caption
	}
	return entitiesToHTML(msg.Caption, msg.CaptionEntities)
}

func (h *Bot) saveMessagePhoto(ctx context.Context, b *bot.Bot, msg *models.Message) (string, error) {
	ph := msg.Photo[len(msg.Photo)-1]
	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: ph.FileID})
	if err != nil {
		return "", err
	}
	if f.FileSize > domain.MaxImageBytes {
		return "", fmt.Errorf("too large")
	}
	link := b.FileDownloadLink(f)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, domain.MaxImageBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > domain.MaxImageBytes {
		return "", fmt.Errorf("too large")
	}
	ct := http.DetectContentType(data)
	ext, err := domain.ImageExt(ct)
	if err != nil {
		return "", err
	}
	var id [8]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	name := hex.EncodeToString(id[:]) + ext
	if err := h.images.Save(ctx, name, bytes.NewReader(data)); err != nil {
		return "", err
	}
	return name, nil
}

func (h *Bot) sendMedia(ctx context.Context, b *bot.Bot, chatID int64, text, imageName string, markup models.ReplyMarkup) {
	if imageName == "" {
		h.send(ctx, b, chatID, text, markup)
		return
	}
	if !h.images.Exists(imageName) {
		h.send(ctx, b, chatID, config.PhotoMissing+"\n\n"+text, markup)
		return
	}
	req, _ := ctx.Value(uiKey{}).(uiReq)
	markup = ensureMenu(markup, req.noMenu)
	caption := clip(text, 1000)
	if req.tgID != 0 && !req.fresh {
		if h.tryEditPhoto(ctx, b, req.tgID, chatID, caption, imageName, markup) {
			return
		}
	}
	msg, err := h.postPhoto(ctx, b, chatID, caption, imageName, markup)
	if err != nil {
		slog.Error("Failed to send photo", "error", err)
		h.send(ctx, b, chatID, config.PhotoMissing+"\n\n"+text, markup)
		return
	}
	if req.tgID != 0 {
		h.bindUIMedia(req.tgID, chatID, msg.ID, imageName)
	}
}

func (h *Bot) tryEditPhoto(ctx context.Context, b *bot.Bot, tgID, chatID int64, caption, imageName string, markup models.ReplyMarkup) bool {
	ref, ok := h.lastUI(tgID)
	if !ok || ref.msgID == 0 || ref.chatID != chatID || ref.image == "" {
		return false
	}
	if ref.image == imageName {
		err := h.editPhotoCaption(ctx, b, chatID, ref.msgID, caption, markup)
		if err == nil || isNotModified(err) {
			return true
		}
		slog.Info("Edit caption failed", "error", err)
		return false
	}
	err := h.editPhotoMedia(ctx, b, chatID, ref.msgID, caption, imageName, markup)
	if err == nil || isNotModified(err) {
		h.bindUIMedia(tgID, chatID, ref.msgID, imageName)
		return true
	}
	slog.Info("Edit media failed", "error", err)
	return false
}

func (h *Bot) editPhotoCaption(ctx context.Context, b *bot.Bot, chatID int64, msgID int, caption string, markup models.ReplyMarkup) error {
	_, err := b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Caption:     messageHTML(caption),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: markup,
	})
	if err == nil || isNotModified(err) {
		return err
	}
	_, err = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Caption:     caption,
		ReplyMarkup: markup,
	})
	return err
}

func (h *Bot) editPhotoMedia(ctx context.Context, b *bot.Bot, chatID int64, msgID int, caption, imageName string, markup models.ReplyMarkup) error {
	f, err := os.Open(h.images.Path(imageName))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = b.EditMessageMedia(ctx, &bot.EditMessageMediaParams{
		ChatID:    chatID,
		MessageID: msgID,
		Media: &models.InputMediaPhoto{
			Media:           "attach://" + imageName,
			Caption:         messageHTML(caption),
			ParseMode:       models.ParseModeHTML,
			MediaAttachment: f,
		},
		ReplyMarkup: markup,
	})
	if err == nil || isNotModified(err) {
		return err
	}
	_, _ = f.Seek(0, 0)
	_, err = b.EditMessageMedia(ctx, &bot.EditMessageMediaParams{
		ChatID:    chatID,
		MessageID: msgID,
		Media: &models.InputMediaPhoto{
			Media:           "attach://" + imageName,
			Caption:         caption,
			MediaAttachment: f,
		},
		ReplyMarkup: markup,
	})
	return err
}

func (h *Bot) postPhoto(ctx context.Context, b *bot.Bot, chatID int64, caption, imageName string, markup models.ReplyMarkup) (*models.Message, error) {
	f, err := os.Open(h.images.Path(imageName))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Photo:       &models.InputFileUpload{Filename: imageName, Data: f},
		Caption:     messageHTML(caption),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: markup,
	})
	if err == nil {
		return msg, nil
	}
	_, _ = f.Seek(0, 0)
	return b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Photo:       &models.InputFileUpload{Filename: imageName, Data: f},
		Caption:     caption,
		ReplyMarkup: markup,
	})
}
