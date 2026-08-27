package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
)

const telegramHTTPTimeout = 30 * time.Second

var telegramHTTP = &http.Client{Timeout: telegramHTTPTimeout}

func downloadTelegramFile(ctx context.Context, b *bot.Bot, fileID string, maxBytes int64) ([]byte, error) {
	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, err
	}
	if int64(f.FileSize) > maxBytes {
		return nil, fmt.Errorf("too large")
	}
	link := b.FileDownloadLink(f)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := telegramHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("too large")
	}
	return data, nil
}
