package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port              int
	AuctionServiceURL string
	BidServiceURL     string
	RabbitMQURL       string
	LogLevel          string
}

func Load() (*Config, error) {
	port := getEnvInt("PORT", 0)
	if port == 0 {
		port = getEnvInt("BFF_PORT", 8080)
	}

	auctionServiceURL := getEnv("AUCTION_SERVICE_URL", "")
	bidServiceURL := getEnv("BID_SERVICE_URL", "")
	rabbitMQURL := getEnv("RABBITMQ_URL", "")
	logLevel := getEnv("LOG_LEVEL", "info")

	if auctionServiceURL == "" {
		return nil, fmt.Errorf("AUCTION_SERVICE_URL is required")
	}
	if bidServiceURL == "" {
		return nil, fmt.Errorf("BID_SERVICE_URL is required")
	}
	if rabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}

	return &Config{
		Port:              port,
		AuctionServiceURL: auctionServiceURL,
		BidServiceURL:     bidServiceURL,
		RabbitMQURL:       rabbitMQURL,
		LogLevel:          logLevel,
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
