package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/config"
	bidhttp "github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/http"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/lock"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/mq"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/observability"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/repo"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/service"
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
	logger, err := observability.NewLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	// 3. Connect Postgres (Bun) + ping + run migrations.
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

	// 4. Connect Redis (lock manager).
	lockMgr, err := lock.NewRedisLockManager(cfg.RedisURL, logger)
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis lock manager connected")

	// 5. Connect RabbitMQ (publisher — dedicated connection).
	publisher, err := mq.NewBidPublisher(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Fatal("failed to connect RabbitMQ publisher", zap.Error(err))
	}
	logger.Info("RabbitMQ publisher connected")

	// 6. Build repos.
	bidRepo := repo.NewPostgresBidRepo(db)
	snapshotRepo := repo.NewPostgresSnapshotRepo(db)

	// 7. Connect RabbitMQ (consumer — separate connection).
	consumer, err := mq.NewAuctionEventConsumer(cfg.RabbitMQURL, snapshotRepo, logger)
	if err != nil {
		logger.Fatal("failed to connect RabbitMQ consumer", zap.Error(err))
	}
	logger.Info("RabbitMQ consumer connected")

	// 8. Build service → handler → router.
	bidSvc := service.NewBidService(bidRepo, snapshotRepo, lockMgr, publisher, logger)
	handler := bidhttp.NewBidHandler(bidSvc, logger)
	router := bidhttp.NewRouter(handler, logger)

	// 9. Start consumer in a goroutine.
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	consumerErrCh := make(chan error, 1)
	go func() {
		if err := consumer.Start(consumerCtx); err != nil {
			consumerErrCh <- err
		}
		close(consumerErrCh)
	}()
	logger.Info("auction event consumer started")

	// 10. Start HTTP server on configured port.
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("bid-service HTTP server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// 11. Wait for SIGINT / SIGTERM.
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

	logger.Info("shutting down bid-service...")

	// 12. Graceful shutdown.

	// Cancel consumer context first.
	consumerCancel()

	// Close publisher.
	if err := publisher.Close(); err != nil {
		logger.Error("error closing RabbitMQ publisher", zap.Error(err))
	}

	// Close consumer connection.
	if err := consumer.Close(); err != nil {
		logger.Error("error closing RabbitMQ consumer", zap.Error(err))
	}

	// Close Redis lock manager.
	if err := lockMgr.Close(); err != nil {
		logger.Error("error closing Redis lock manager", zap.Error(err))
	}

	// Close Postgres (already deferred, but explicit for ordering).
	if err := db.Close(); err != nil {
		logger.Error("error closing database", zap.Error(err))
	}

	// Stop HTTP server with a graceful timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error during HTTP server shutdown", zap.Error(err))
	}

	logger.Info("bid-service stopped")
}
