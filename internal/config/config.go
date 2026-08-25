package config

import (
	"errors"
	"os"
)

var StartMessage = `Привет! 👋

Я бот для интервального повторения — что-то вроде Anki прямо в Telegram.

Пока я только учусь, но скоро здесь появятся:
• создание колод
• добавление карточек
• умные повторения

Используй /help, чтобы посмотреть доступные команды.`

var HelpMessage = `тут короче будет хэлп инфа`

type Config struct {
	TGToken     string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		TGToken:     os.Getenv("TG_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.TGToken == "" {
		return Config{}, errors.New("TG_TOKEN is not set")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is not set")
	}
	return cfg, nil
}
