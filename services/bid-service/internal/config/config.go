package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        int
	DatabaseURL string
	RabbitMQURL string
	RedisURL    string
	LogLevel    string
	LockTTL     time.Duration
}

func Load() (*Config, error) {
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("BID_SERVICE_PORT", 8082)
	}
	dbURL := getEnv("DATABASE_URL", "")
	rabbitURL := getEnv("RABBITMQ_URL", "")
	redisURL := getEnv("REDIS_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")
	lockTTLSec := getEnvInt("BID_LOCK_TTL_SEC", 5)

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if rabbitURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		RabbitMQURL: rabbitURL,
		RedisURL:    redisURL,
		LogLevel:    logLevel,
		LockTTL:     time.Duration(lockTTLSec) * time.Second,
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
