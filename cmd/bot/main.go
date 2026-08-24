package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	tg "github.com/drenk83/teleanki/internal/telegram"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found")
	}

	tgToken := os.Getenv("TG_TOKEN")
	if tgToken == "" {
		slog.Error("TG_TOKEN is not set")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b, err := tg.CreateBot(tgToken)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	slog.Info("Bot is starting...")
	b.Start(ctx)
}
