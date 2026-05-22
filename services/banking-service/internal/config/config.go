package config

import (
	"fmt"
	"os"
	"strconv"
)

type config struct {
	Port        int
	DatabaseURL string
	RabbitMQURL string
	LogLevel    string
}

func Load() (*config, error) {
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("BANKING_SERVICE_PORT", 8084)
	}
	dbURL := getEnv("DATABASE_URL", "")
	rabbitURL := getEnv("RABBITMQ_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if rabbitURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}

	return &config{
		Port:        port,
		DatabaseURL: dbURL,
		RabbitMQURL: rabbitURL,
		LogLevel:    logLevel,
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
