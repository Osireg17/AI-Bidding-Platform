package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port          int
	DatabaseURL   string
	RabbitMQURL   string
	BidServiceURL string
	LogLevel      string
	GeminiAPIKey  string
}

func Load() (*Config, error) {
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("BOT_SERVICE_PORT", 8083)
	}
	dbURL := getEnv("DATABASE_URL", "")
	rabbitURL := getEnv("RABBITMQ_URL", "")
	bidServiceURL := getEnv("BID_SERVICE_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")
	geminiAPIKey := getEnv("GEMINI_API_KEY", "")

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if rabbitURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if bidServiceURL == "" {
		return nil, fmt.Errorf("BID_SERVICE_URL is required")
	}
	if geminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		RabbitMQURL:   rabbitURL,
		BidServiceURL: bidServiceURL,
		LogLevel:      logLevel,
		GeminiAPIKey:  geminiAPIKey,
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return intVal
}
