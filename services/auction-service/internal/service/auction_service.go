package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"go.uber.org/zap"
)

// === CONTEXT ===
// Purpose: Application service that orchestrates auction lifecycle operations.
// Sits between HTTP handlers and domain/infrastructure. Contains use-case logic.
// Reference: domain/ports.go for the interfaces this service depends on.
//
// === DEPENDENCIES ===
// domain.AuctionRepository — persistence (injected)
// domain.EventPublisher — event publishing (injected)
// zap.Logger — structured logging (injected)
//
// === DATA / STATE ===
// AuctionService is stateless — all state lives in the DB.
// Created once at startup, shared across request goroutines (thread-safe via DB).

// AuctionService orchestrates auction use cases.
type AuctionService struct {
	repo      domain.AuctionRepository
	publisher domain.EventPublisher
	logger    *zap.Logger
}

// NewAuctionService creates an AuctionService with its dependencies.
func NewAuctionService(repo domain.AuctionRepository, publisher domain.EventPublisher, logger *zap.Logger) *AuctionService {
	return &AuctionService{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// === BEHAVIOR: CreateAuction ===
// Input: context, title, description, startPrice, duration
// Output: *Auction or error
// Logic:
//   CREATE auction domain object (validates inputs)
//   ACTIVATE the auction immediately (pending -> active)
//   PERSIST to database
//   PUBLISH auction.created event
//   RETURN the auction

func (s *AuctionService) CreateAuction(ctx context.Context, title, description string, startPrice float64, duration time.Duration) (*domain.Auction, error) {
	auction, err := domain.NewAuction(title, description, startPrice, duration)
	if err != nil {
		return nil, err
	}

	if err := auction.Activate(); err != nil {
		return nil, fmt.Errorf("failed to activate new auction: %w", err)
	}

	if err := s.repo.Create(ctx, auction); err != nil {
		return nil, fmt.Errorf("failed to persist auction: %w", err)
	}

	if err := s.publisher.PublishAuctionCreated(ctx, auction); err != nil {
		s.logger.Error("failed to publish auction.created event",
			zap.Int64("auction_id", auction.ID),
			zap.Error(err),
		)
		// Don't fail the creation — the auction is persisted. Event publishing is best-effort for now.
	}

	s.logger.Info("auction created",
		zap.Int64("auction_id", auction.ID),
		zap.String("title", auction.Title),
		zap.Time("end_time", auction.EndTime),
	)
	return auction, nil
}

// === BEHAVIOR: GetAuction ===
// Input: context, auction ID
// Output: *Auction or error (ErrAuctionNotFound if missing)

func (s *AuctionService) GetAuction(ctx context.Context, id int64) (*domain.Auction, error) {
	return s.repo.GetByID(ctx, id)
}

// === BEHAVIOR: ListAuctions ===
// Input: context
// Output: slice of all auctions

func (s *AuctionService) ListAuctions(ctx context.Context) ([]*domain.Auction, error) {
	return s.repo.List(ctx)
}

// === BEHAVIOR: ProcessExpiredAuctions ===
// Input: context
// Output: error
// Logic:
//   FIND all active/ending_soon auctions past their end_time
//   FOR EACH expired auction:
//     CLOSE the auction (no winner lookup for now — MVP placeholder)
//     UPDATE in database
//     PUBLISH auction.ended event
//   LOG results
// Edge Cases: no expired auctions (normal, skip silently)

func (s *AuctionService) ProcessExpiredAuctions(ctx context.Context) error {
	expired, err := s.repo.FindExpiredActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to find expired auctions: %w", err)
	}

	for _, auction := range expired {
		// MVP: close without winner. Winner determination via REST call to bid-service will be added later.
		if err := auction.Close(0, 0); err != nil {
			s.logger.Error("failed to close expired auction",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
			continue
		}

		if err := s.repo.Update(ctx, auction); err != nil {
			s.logger.Error("failed to update closed auction",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
			continue
		}

		if err := s.publisher.PublishAuctionEnded(ctx, auction); err != nil {
			s.logger.Error("failed to publish auction.ended event",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
		}

		s.logger.Info("auction closed",
			zap.Int64("auction_id", auction.ID),
			zap.String("final_status", "unsold"),
		)
	}
	return nil
}

// === BEHAVIOR: ProcessEndingSoonAuctions ===
// Input: context, ending-soon threshold in seconds
// Output: error
// Logic:
//   FIND active auctions within the ending-soon window
//   FOR EACH auction:
//     MARK as ending soon (status transition)
//     UPDATE in database
//     PUBLISH auction.ending_soon event
//   LOG results

func (s *AuctionService) ProcessEndingSoonAuctions(ctx context.Context, thresholdSeconds int) error {
	auctions, err := s.repo.FindEndingSoon(ctx, thresholdSeconds)
	if err != nil {
		return fmt.Errorf("failed to find ending-soon auctions: %w", err)
	}

	for _, auction := range auctions {
		if err := auction.MarkEndingSoon(); err != nil {
			s.logger.Error("failed to mark auction as ending soon",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
			continue
		}

		if err := s.repo.Update(ctx, auction); err != nil {
			s.logger.Error("failed to update ending-soon auction",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
			continue
		}

		if err := s.publisher.PublishAuctionEndingSoon(ctx, auction); err != nil {
			s.logger.Error("failed to publish auction.ending_soon event",
				zap.Int64("auction_id", auction.ID),
				zap.Error(err),
			)
		}

		s.logger.Info("auction marked ending soon",
			zap.Int64("auction_id", auction.ID),
			zap.Time("end_time", auction.EndTime),
		)
	}
	return nil
}
