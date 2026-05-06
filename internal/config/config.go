package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GRPCPort               string
	DatabaseURL            string
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
	CacheTTL               time.Duration
	KafkaBrokers           []string
	KafkaTopicChatMessages string
}

func Load() (Config, error) {
	redisDB := 0
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, errors.New("REDIS_DB must be an integer")
		}
		redisDB = parsed
	}

	cacheTTL := 20 * time.Minute
	if raw := os.Getenv("CACHE_TTL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, errors.New("CACHE_TTL must be a valid duration, for example 20m")
		}
		cacheTTL = parsed
	}

	cfg := Config{
		GRPCPort:               getEnvOrDefault("GRPC_PORT", "50051"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		RedisAddr:              os.Getenv("REDIS_ADDR"),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		RedisDB:                redisDB,
		CacheTTL:               cacheTTL,
		KafkaBrokers:           parseCSV(os.Getenv("KAFKA_BROKERS")),
		KafkaTopicChatMessages: getEnvOrDefault("KAFKA_TOPIC_CHAT_MESSAGES", "chat.messages"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		val := strings.TrimSpace(part)
		if val != "" {
			result = append(result, val)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func getEnvOrDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
