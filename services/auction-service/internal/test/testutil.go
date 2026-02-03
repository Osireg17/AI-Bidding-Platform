package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/repo"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
	"testing"
)

func NewTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		dsn = buildPostgresDSN()
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	if err := repo.RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func CleanupDB(t *testing.T, db *bun.DB) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE auctions RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("failed to cleanup database: %v", err)
	}
}

func NewTestRabbitMQ(t *testing.T) (*amqp.Connection, *amqp.Channel) {
	t.Helper()

	url := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("failed to open rabbitmq channel: %v", err)
	}

	return conn, ch
}

func NewTestLogger() *zap.Logger {
	return zap.NewNop()
}

func CreateTestAuction(overrides *domain.Auction) *domain.Auction {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	auction := &domain.Auction{
		ID:           1,
		Title:        "Test Auction",
		Description:  "Test Description",
		StartPrice:   10.0,
		CurrentPrice: 10.0,
		Status:       domain.StatusPending,
		WinnerBotID:  0,
		StartTime:    now,
		EndTime:      now.Add(5 * time.Minute),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if overrides == nil {
		return auction
	}

	if overrides.ID != 0 {
		auction.ID = overrides.ID
	}
	if overrides.Title != "" {
		auction.Title = overrides.Title
	}
	if overrides.Description != "" {
		auction.Description = overrides.Description
	}
	if overrides.StartPrice != 0 {
		auction.StartPrice = overrides.StartPrice
	}
	if overrides.CurrentPrice != 0 {
		auction.CurrentPrice = overrides.CurrentPrice
	}
	if overrides.Status != "" {
		auction.Status = overrides.Status
	}
	if overrides.WinnerBotID != 0 {
		auction.WinnerBotID = overrides.WinnerBotID
	}
	if !overrides.StartTime.IsZero() {
		auction.StartTime = overrides.StartTime
	}
	if !overrides.EndTime.IsZero() {
		auction.EndTime = overrides.EndTime
	}
	if !overrides.CreatedAt.IsZero() {
		auction.CreatedAt = overrides.CreatedAt
	}
	if !overrides.UpdatedAt.IsZero() {
		auction.UpdatedAt = overrides.UpdatedAt
	}

	return auction
}

func buildPostgresDSN() string {
	user := getEnv("POSTGRES_USER", "bidding")
	pass := getEnv("POSTGRES_PASSWORD", "bidding_dev")
	db := getEnv("POSTGRES_DB", "bidding_platform")
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
