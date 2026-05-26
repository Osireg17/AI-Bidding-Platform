package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/bankingclient"
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

	// 3. Connect Postgres + ping + run migrations.
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

	bankingClient := bankingclient.NewBankingServiceClient(cfg.BankingServiceURL, logger)
	logger.Info("banking service client initialised", zap.String("url", cfg.BankingServiceURL))

	// 6. Init repo.
	botBidRepo := repo.NewPostgresBotBidRepo(db)

	// 7. Init 4 bot agents.
	initCtx := context.Background()
	bots := make([]mq.BotEvaluator, 0, len(domain.AllBots))
	for _, bot := range domain.AllBots {
		a, err := agent.NewBotAgent(initCtx, bot, cfg.GeminiAPIKey, bidClient, bankingClient, botBidRepo, logger)
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

	// 9. Start the consumer under a window-aware scheduler.
	// Active window: 09:00–21:00 UTC daily. Outside this window the consumer
	// is paused and RabbitMQ messages queue up for processing when it resumes.
	appCtx, appCancel := context.WithCancel(context.Background())
	schedulerErrCh := make(chan error, 1)
	go func() {
		schedulerErrCh <- runScheduled(appCtx, consumer, logger)
		close(schedulerErrCh)
	}()
	logger.Info("bot event consumer scheduler started")

	// 10. Wait for SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	case err := <-schedulerErrCh:
		if err != nil {
			logger.Error("scheduler exited with error, initiating shutdown", zap.Error(err))
		}
	}

	logger.Info("shutting down bot-service...")

	// 11. Graceful shutdown.
	appCancel()

	if err := consumer.Close(); err != nil {
		logger.Error("error closing RabbitMQ consumer", zap.Error(err))
	}

	if err := db.Close(); err != nil {
		logger.Error("error closing database", zap.Error(err))
	}

	logger.Info("bot-service stopped")
}

// runScheduled is the window-aware scheduler for the bot event consumer.
func runScheduled(ctx context.Context, consumer *mq.BotEventConsumer, logger *zap.Logger) error {
	const (
		windowStart = 9  // hour UTC
		windowEnd   = 21 // hour UTC
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		now := time.Now().UTC()
		inWindow := now.Hour() >= windowStart && now.Hour() < windowEnd

		if inWindow {
			logger.Info("active window started, starting consumer")
			consumerCtx, consumerCancel := context.WithCancel(ctx)

			consumerErrCh := make(chan error, 1)
			go func() {
				consumerErrCh <- consumer.Start(consumerCtx)
			}()

			// Calculate duration until window closes (21:00 UTC today).
			windowCloseToday := time.Date(now.Year(), now.Month(), now.Day(), windowEnd, 0, 0, 0, time.UTC)
			durationUntilClose := time.Until(windowCloseToday)

			select {
			case <-time.After(durationUntilClose):
				logger.Info("active window ended, pausing consumer")
				consumerCancel()
				<-consumerErrCh // Drain the channel to allow the goroutine to exit and avoid leaks
			case err := <-consumerErrCh:
				consumerCancel()
				if err != nil {
					return fmt.Errorf("consumer error: %w", err)
				}
			case <-ctx.Done():
				consumerCancel()
				// Drain the channel (if the goroutine already finished or finishes shortly)
				// without blocking the shutdown process indefinitely.
				select {
				case <-consumerErrCh:
				case <-time.After(1 * time.Second):
				}
				return nil
			}
		} else {
			logger.Info("outside active window, consumer paused")

			// Calculate duration until window opens (09:00 UTC).
			// If current hour >= windowEnd, next open is 09:00 tomorrow.
			nextOpen := time.Date(now.Year(), now.Month(), now.Day(), windowStart, 0, 0, 0, time.UTC)
			if now.Hour() >= windowEnd {
				nextOpen = nextOpen.Add(24 * time.Hour)
			}
			durationUntilOpen := time.Until(nextOpen)

			select {
			case <-time.After(durationUntilOpen):
				// loop back around — inWindow will now be true
			case <-ctx.Done():
				return nil
			}
		}
	}
}
