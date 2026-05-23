package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/config"
	bankinghttp "github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/http"
	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/mq"
	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/repo"
	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

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

	walletRepo := repo.NewPostgresWalletRepo(db)
	itemRepo := repo.NewPostgresItemRepo(db)

	svc := service.NewBankingService(db, walletRepo, itemRepo, logger)

	if err := svc.SeedWallets(context.Background()); err != nil {
		logger.Fatal("failed to seed wallets", zap.Error(err))
	}
	logger.Info("wallets seeded")

	consumer, err := mq.NewBankingEventConsumer(cfg.RabbitMQURL, svc, logger)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer consumer.Close()

	handler := bankinghttp.NewBankingHandler(svc, logger)
	router := bankinghttp.NewRouter(handler, logger)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	go func() {
		if err := consumer.Start(consumerCtx); err != nil {
			logger.Error("consumer stopped with error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("banking-service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down banking-service...")

	consumerCancel()
	if err := srv.Close(); err != nil {
		logger.Error("error closing HTTP server", zap.Error(err))
	}
	logger.Info("banking-service stopped")
}
