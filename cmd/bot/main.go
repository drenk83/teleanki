package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/drenk83/teleanki/internal/storage/postgres"
	tg "github.com/drenk83/teleanki/internal/telegram"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found")
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	users := postgres.NewUserRepository(db)
	decks := postgres.NewDeckRepository(db)
	cards := postgres.NewCardRepository(db)
	reviews := postgres.NewReviewRepository(db)

	b, err := tg.CreateBot(cfg.TGToken, users, decks, cards, reviews)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	slog.Info("Bot is starting...")
	b.Start(ctx)
}
