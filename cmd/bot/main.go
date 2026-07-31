package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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

	opts := []bot.Option{
		bot.WithMiddlewares(loggingMiddleware),
	}

	b, err := bot.New(tgToken, opts...)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, helpHandler)

	slog.Info("Bot is starting...")
	b.Start(ctx)
}

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   config.StartMessage,
	})
	if err != nil {
		slog.Error("Failed to send message", "error", err)
		return
	}
	slog.Info("Send message",
		"type_message", "start_message",
		"user_id", update.Message.From.ID,
		"username", update.Message.From.Username,
	)
}

func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   config.HelpMessage,
	})
	if err != nil {
		slog.Error("Failed to send message", "error", err)
		return
	}
	slog.Info("Send message",
		"type_message", "help_message",
		"user_id", update.Message.From.ID,
		"username", update.Message.From.Username,
	)
}

func loggingMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil && update.Message.Text != "" {
			slog.Info("Received message",
				"user_id", update.Message.From.ID,
				"username", update.Message.From.Username,
				"text", update.Message.Text,
			)
		}
		next(ctx, b, update)
	}
}
