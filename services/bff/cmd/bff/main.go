package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/broadcaster"
	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/client"
	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/config"
	bffhttp "github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/http"
	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/mq"
	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/store"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	sharedlogger "github.com/Osireg17/AI-Bidding-Platform/shared/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := sharedlogger.NewLogger(cfg.LogLevel, "bff")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	stateStore := store.NewInMemoryStateStore(logger)
	eventBroadcaster := broadcaster.NewBroadcaster(logger)

	// Start the MQ consumer before hydration so no events are lost
	// between the hydration HTTP calls and consumer start.
	consumer, err := mq.NewBFFEventConsumer(cfg.RabbitMQURL, stateStore, eventBroadcaster, logger)
	if err != nil {
		logger.Fatal("failed to create BFF event consumer", zap.Error(err))
	}
	defer consumer.Close()

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	go func() {
		if err := consumer.Start(consumerCtx); err != nil {
			logger.Error("BFF event consumer stopped with error", zap.Error(err))
		}
	}()

	// Hydrate state from upstream services.
	auctionClient := client.NewAuctionServiceClient(cfg.AuctionServiceURL, logger)
	bidClient := client.NewBidServiceClient(cfg.BidServiceURL, logger)

	hydrate(context.Background(), stateStore, auctionClient, bidClient, logger)

	handler := bffhttp.NewBFFHandler(stateStore, eventBroadcaster, logger)
	router := bffhttp.NewRouter(handler, logger, cfg.AllowedOrigin)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("bff starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down bff...")
	consumerCancel()
	if err := srv.Close(); err != nil {
		logger.Error("error closing HTTP server", zap.Error(err))
	}
	logger.Info("bff stopped")
}

// hydrate loads the current auction and its highest bid into the state store on startup.
// Errors are logged but do not fatal — the store starts empty and fills via MQ events.
func hydrate(
	ctx context.Context,
	stateStore *store.InMemoryStateStore,
	auctionClient *client.AuctionServiceClient,
	bidClient *client.BidServiceClient,
	logger *zap.Logger,
) {
	auction, err := auctionClient.GetActiveAuction(ctx)
	if err != nil {
		logger.Warn("hydration: failed to fetch active auction", zap.Error(err))
		return
	}
	if auction == nil {
		logger.Info("hydration: no active auction found")
		return
	}

	stateStore.ApplyAuctionCreated(events.AuctionCreatedPayload{
		AuctionID:   auction.ID,
		Title:       auction.Title,
		Description: auction.Description,
		StartPrice:  auction.StartPrice,
		EndTime:     auction.EndTime.Format("2006-01-02T15:04:05Z07:00"),
	})

	if auction.Status == "ending_soon" {
		stateStore.ApplyAuctionEndingSoon(events.AuctionEndingSoonPayload{
			AuctionID: auction.ID,
		})
	}

	botID, amount, err := bidClient.GetHighestBid(ctx, auction.ID)
	if err != nil {
		logger.Warn("hydration: failed to fetch highest bid", zap.Int64("auction_id", auction.ID), zap.Error(err))
		return
	}

	if amount > 0 {
		stateStore.ApplyBidPlaced(events.BidPlacedPayload{
			AuctionID: auction.ID,
			BotID:     botID,
			BidAmount: amount,
		})
	}

	logger.Info("hydration complete",
		zap.Int64("auction_id", auction.ID),
		zap.String("status", auction.Status),
		zap.Float64("current_price", auction.CurrentPrice),
	)
}
