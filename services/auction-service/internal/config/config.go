package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// === CONTEXT ===
// Purpose: Load environment variables into a typed Config struct.
// Single source of truth for all auction-service configuration.
//
// === BEHAVIOR: Load ===
// Input: environment variables (from OS or .env)
// Output: *Config with parsed values, or error if required vars are missing
// Logic:
//   READ each env var with fallback defaults for dev
//   PARSE numeric values (port, thresholds, intervals)
//   VALIDATE required fields (DatabaseURL, RabbitMQURL)
//   RETURN populated Config

// Config holds all configuration for the auction service.
type Config struct {
	Port                 int
	DatabaseURL          string
	RabbitMQURL          string
	LogLevel             string
	EndingSoonThreshold  time.Duration
	AuctionCheckInterval time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	// Railway injects PORT; fall back to AUCTION_SERVICE_PORT for local dev.
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("AUCTION_SERVICE_PORT", 8081)
	}
	dbURL := getEnv("DATABASE_URL", "")
	rabbitURL := getEnv("RABBITMQ_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")
	endingSoonSec := getEnvInt("ENDING_SOON_THRESHOLD_SEC", 30)
	checkIntervalSec := getEnvInt("AUCTION_CHECK_INTERVAL_SEC", 5)

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if rabbitURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}

	return &Config{
		Port:                 port,
		DatabaseURL:          dbURL,
		RabbitMQURL:          rabbitURL,
		LogLevel:             logLevel,
		EndingSoonThreshold:  time.Duration(endingSoonSec) * time.Second,
		AuctionCheckInterval: time.Duration(checkIntervalSec) * time.Second,
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
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}
