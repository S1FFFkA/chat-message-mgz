package config

import (
	"errors"
	"os"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		GRPCPort:    getEnvOrDefault("GRPC_PORT", "50051"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
