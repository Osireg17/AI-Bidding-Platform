package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// === CONTEXT ===
// Purpose: Background scheduler that periodically checks for auction store transitions.
// Runs a ticker that calls ProcessExpiredAuctions and ProcessEndingSoonAuctions.
// Reference: service/auction_service.go for the methods being called.
//
// === DEPENDENCIES ===
// AuctionService — the service layer methods for processing auctions
// zap.Logger — structured logging
//
// === BEHAVIOR: StartScheduler ===
// Input: context (for cancellation), AuctionService, check interval, ending-soon threshold
// Output: none (runs as a goroutine until context is cancelled)
// Logic:
//   CREATE a ticker with the check interval
//   LOOP on ticker ticks:
//     CALL ProcessExpiredAuctions
//     CALL ProcessEndingSoonAuctions
//   STOP when context is cancelled
// Edge Cases: context cancelled mid-tick (operations use context, will abort)

// StartScheduler runs the auction lifecycle checker in a background goroutine.
// It returns immediately. Cancel the context to stop the scheduler.
func StartScheduler(ctx context.Context, svc *AuctionService, interval time.Duration, endingSoonThreshold time.Duration, logger *zap.Logger) {
	ticker := time.NewTicker(interval)

	go func() {
		logger.Info("auction scheduler started",
			zap.Duration("interval", interval),
			zap.Duration("ending_soon_threshold", endingSoonThreshold),
		)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("auction scheduler stopped")
				return
			case <-ticker.C:
				thresholdSeconds := int(endingSoonThreshold.Seconds())

				if err := svc.ProcessEndingSoonAuctions(ctx, thresholdSeconds); err != nil {
					logger.Error("scheduler: failed to process ending-soon auctions", zap.Error(err))
				}

				if err := svc.ProcessExpiredAuctions(ctx); err != nil {
					logger.Error("scheduler: failed to process expired auctions", zap.Error(err))
				}
			}
		}
	}()
}
