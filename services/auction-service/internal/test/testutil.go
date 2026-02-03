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
	"go.uber.org/zap/zaptest"
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
	t.Cleanup(func() {
		_ = db.Close()
	})

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

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if _, err := db.ExecContext(ctx, "TRUNCATE TABLE auctions RESTART IDENTITY CASCADE"); err != nil {
    t.Fatalf("failed to cleanup database: %v", err)
}
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

	t.Cleanup(func() {
		_ = ch.Close()
		_ = conn.Close()
	})

	return conn, ch
}

func NewTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zaptest.NewLogger(t)
}

type AuctionOption func(*domain.Auction)

func CreateTestAuction(opts ...AuctionOption) *domain.Auction {
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

	for _, opt := range opts {
		opt(auction)
	}

	return auction
}

func WithAuctionID(id int64) AuctionOption {
	return func(a *domain.Auction) {
		a.ID = id
	}
}

func WithAuctionTitle(title string) AuctionOption {
	return func(a *domain.Auction) {
		a.Title = title
	}
}

func WithAuctionDescription(description string) AuctionOption {
	return func(a *domain.Auction) {
		a.Description = description
	}
}

func WithAuctionStartPrice(price float64) AuctionOption {
	return func(a *domain.Auction) {
		a.StartPrice = price
	}
}

func WithAuctionCurrentPrice(price float64) AuctionOption {
	return func(a *domain.Auction) {
		a.CurrentPrice = price
	}
}

func WithAuctionStatus(status domain.AuctionStatus) AuctionOption {
	return func(a *domain.Auction) {
		a.Status = status
	}
}

func WithAuctionWinnerBotID(id int64) AuctionOption {
	return func(a *domain.Auction) {
		a.WinnerBotID = id
	}
}

func WithAuctionStartTime(t time.Time) AuctionOption {
	return func(a *domain.Auction) {
		a.StartTime = t
	}
}

func WithAuctionEndTime(t time.Time) AuctionOption {
	return func(a *domain.Auction) {
		a.EndTime = t
	}
}

func WithAuctionCreatedAt(t time.Time) AuctionOption {
	return func(a *domain.Auction) {
		a.CreatedAt = t
	}
}

func WithAuctionUpdatedAt(t time.Time) AuctionOption {
	return func(a *domain.Auction) {
		a.UpdatedAt = t
	}
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
