package config

import (
	"errors"
	"os"
)

type Config struct {
	TGToken     string
	DatabaseURL string
	ImagesDir   string
}

func Load() (Config, error) {
	cfg := Config{
		TGToken:     os.Getenv("TG_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ImagesDir:   os.Getenv("IMAGES_DIR"),
	}
	if cfg.ImagesDir == "" {
		cfg.ImagesDir = "data/images"
	}
	if cfg.TGToken == "" {
		return Config{}, errors.New("TG_TOKEN is not set")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is not set")
	}
	return cfg, nil
}
