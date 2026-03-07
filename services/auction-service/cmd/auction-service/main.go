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

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/bidclient"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/config"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
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

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Initialize logger.
	logger, err := observability.NewLogger(cfg.LogLevel)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {

		}
	}(logger)

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
	bidClient := bidclient.NewBidServiceClient(cfg.BidServiceURL, logger)

	// Wire the auction agent (generates new items after each auction closes).
	// Key is passed directly via ClientConfig in agent.Generate — no os.Setenv needed.
	var auctionAgent *agent.AuctionAgent
	if cfg.GeminiAPIKey != "" {
		auctionAgent = agent.NewAuctionAgent(cfg.GeminiAPIKey, logger)
		logger.Info("auction agent initialised")
	} else {
		logger.Warn("GEMINI_API_KEY not set — auction agent disabled, no auto-creation after auctions end")
	}

	auctionSvc := service.NewAuctionService(auctionRepo, publisher, bidClient, auctionAgent, logger)
	handler := auctionhttp.NewAuctionHandler(auctionSvc, logger)
	router := auctionhttp.NewRouter(handler, logger)

	// Bootstrap: create an initial auction if none is currently active.
	if auctionAgent != nil {
		auctions, err := auctionSvc.ListAuctions(context.Background())
		if err != nil {
			logger.Fatal("failed to list auctions on startup", zap.Error(err))
		}
		hasActive := false
		for _, a := range auctions {
			if a.Status == domain.StatusActive || a.Status == domain.StatusEndingSoon {
				hasActive = true
				break
			}
		}
		if !hasActive {
			logger.Info("no active auction found, generating initial auction")
			item, err := auctionAgent.Generate(context.Background())
			if err != nil {
				logger.Fatal("failed to generate initial auction item", zap.Error(err))
			}
			if _, err := auctionSvc.CreateAuction(context.Background(), item.Title, item.Description, item.StartPrice, time.Duration(item.DurationSec)*time.Second); err != nil {
				logger.Fatal("failed to create initial auction", zap.Error(err))
			}
		} else {
			logger.Info("active auction already exists, skipping bootstrap")
		}
	}

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
