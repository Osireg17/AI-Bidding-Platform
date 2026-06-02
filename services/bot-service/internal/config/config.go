package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port              int
	DatabaseURL       string
	RabbitMQURL       string
	BankingServiceURL string
	BidServiceURL     string
	LogLevel          string
	GeminiAPIKey      string
}

func Load() (*Config, error) {
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("BOT_SERVICE_PORT", 8083)
	}
	dbURL := getEnv("DATABASE_URL", "")
	rabbitURL := getEnv("RABBITMQ_URL", "")
	bankingServiceURL := getEnv("BANKING_SERVICE_URL", "")
	bidServiceURL := getEnv("BID_SERVICE_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")
	googleAPIKey := getEnv("GOOGLE_API_KEY", "")

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if rabbitURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if bankingServiceURL == "" {
		return nil, fmt.Errorf("BANKING_SERVICE_URL is required")
	}
	if bidServiceURL == "" {
		return nil, fmt.Errorf("BID_SERVICE_URL is required")
	}
	if googleAPIKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is required")
	}

	return &Config{
		Port:              port,
		DatabaseURL:       dbURL,
		RabbitMQURL:       rabbitURL,
		BankingServiceURL: bankingServiceURL,
		BidServiceURL:     bidServiceURL,
		LogLevel:          logLevel,
		GeminiAPIKey:      googleAPIKey,
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
