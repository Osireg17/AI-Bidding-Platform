package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/config"
	auctionhttp "github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/http"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/mq"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/observability"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/repo"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

// === CONTEXT ===
// Purpose: Entry point for the auction-service. Wires all dependencies and starts the server.
//
// === BEHAVIOR: main ===
// Logic:
//   LOAD config from env vars
//   INIT logger
//   CONNECT to Postgres via Bun (pgdriver parses DATABASE_URL)
//   RUN migrations (create tables + indexes)
//   CONNECT to RabbitMQ (publisher)
//   BUILD repository (implements AuctionRepository)
//   BUILD publisher (implements EventPublisher)
//   BUILD auction service (depends on repo + publisher)
//   BUILD HTTP handler (depends on service)
//   BUILD router (depends on handler)
//   START scheduler in background goroutine
//   START HTTP server
//   WAIT for shutdown signal (SIGINT/SIGTERM)
//   GRACEFUL SHUTDOWN: stop scheduler, close publisher, close DB, stop server

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger.
	logger, err := observability.NewLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Connect to Postgres via Bun.
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("failed to ping database", zap.Error(err))
	}
	logger.Info("connected to PostgreSQL")

	// Run database migrations.
	if err := repo.RunMigrations(context.Background(), db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("database migrations applied")

	// Connect to RabbitMQ.
	publisher, err := mq.NewAuctionPublisher(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer publisher.Close()

	// Build dependencies.
	auctionRepo := repo.NewPostgresAuctionRepo(db)
	auctionSvc := service.NewAuctionService(auctionRepo, publisher, logger)
	handler := auctionhttp.NewAuctionHandler(auctionSvc, logger)
	router := auctionhttp.NewRouter(handler, logger)

	// Start scheduler.
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	defer schedulerCancel()
	service.StartScheduler(schedulerCtx, auctionSvc, cfg.AuctionCheckInterval, cfg.EndingSoonThreshold, logger)

	// Start HTTP server.
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("auction-service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down auction-service...")

	// Graceful shutdown.
	schedulerCancel()
	if err := srv.Close(); err != nil {
		logger.Error("error closing HTTP server", zap.Error(err))
	}
	logger.Info("auction-service stopped")
}
