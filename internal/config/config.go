package config

import (
	"errors"
	"os"
)

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
