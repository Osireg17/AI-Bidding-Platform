package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/bidclient"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/config"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/mq"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/repo"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

func main() {
	// 1. Load config.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Init logger.
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck
	logger = logger.With(zap.String("service", "bot-service"))

	// 3. Set GOOGLE_API_KEY for ADK — must be set before NewBotAgent calls gemini.NewModel.
	if err := os.Setenv("GOOGLE_API_KEY", cfg.GeminiAPIKey); err != nil {
		logger.Fatal("failed to set GOOGLE_API_KEY", zap.Error(err))
	}

	// 4. Connect Postgres + ping + run migrations.
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("failed to ping database", zap.Error(err))
	}
	logger.Info("connected to PostgreSQL")

	if err := repo.RunMigrations(context.Background(), db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("database migrations applied")

	// 5. Init bid service client.
	bidClient := bidclient.NewBidServiceClient(cfg.BidServiceURL, logger)
	logger.Info("bid service client initialised", zap.String("url", cfg.BidServiceURL))

	// 6. Init repo.
	botBidRepo := repo.NewPostgresBotBidRepo(db)

	// 7. Init 4 bot agents.
	initCtx := context.Background()
	bots := make([]mq.BotEvaluator, 0, len(domain.AllBots))
	for _, bot := range domain.AllBots {
		a, err := agent.NewBotAgent(initCtx, bot, cfg.GeminiAPIKey, bidClient, botBidRepo, logger)
		if err != nil {
			logger.Fatal("failed to init bot agent",
				zap.String("bot", bot.Name),
				zap.Error(err),
			)
		}
		bots = append(bots, a)
		logger.Info("bot agent initialised", zap.String("bot", bot.Name))
	}

	// 8. Init RabbitMQ consumer.
	consumer, err := mq.NewBotEventConsumer(cfg.RabbitMQURL, bots, logger)
	if err != nil {
		logger.Fatal("failed to connect RabbitMQ consumer", zap.Error(err))
	}
	logger.Info("RabbitMQ consumer connected")

	// 9. Start consumer in a goroutine.
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	consumerErrCh := make(chan error, 1)
	go func() {
		if err := consumer.Start(consumerCtx); err != nil {
			consumerErrCh <- err
		}
		close(consumerErrCh)
	}()
	logger.Info("bot event consumer started")

	// 10. Wait for SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	case err := <-consumerErrCh:
		if err != nil {
			logger.Error("consumer exited with error, initiating shutdown", zap.Error(err))
		}
	}

	logger.Info("shutting down bot-service...")

	// 11. Graceful shutdown.
	consumerCancel()

	if err := consumer.Close(); err != nil {
		logger.Error("error closing RabbitMQ consumer", zap.Error(err))
	}

	if err := db.Close(); err != nil {
		logger.Error("error closing database", zap.Error(err))
	}

	logger.Info("bot-service stopped")
}
