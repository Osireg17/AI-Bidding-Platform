package service

import (
	"context"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"go.uber.org/zap"
)

type BidService struct {
	bidRepo      domain.BidRepository
	snapShotRepo domain.AuctionSnapshotRepository
	lockManager  domain.LockManager
	publisher    domain.EventPublisher
	logger       *zap.Logger
}

func NewBidService(bidRepo domain.BidRepository, snapShotRepo domain.AuctionSnapshotRepository, lockManager domain.LockManager, publisher domain.EventPublisher, logger *zap.Logger) *BidService {
	return &BidService{
		bidRepo:      bidRepo,
		snapShotRepo: snapShotRepo,
		lockManager:  lockManager,
		publisher:    publisher,
		logger:       logger,
	}
}

func (s *BidService) GetWinner(ctx context.Context, auctionID int64) (botID int64, amount float64, err error) {
	return s.bidRepo.GetWinner(ctx, auctionID)
}

func (s *BidService) PlaceBid(ctx context.Context, auctionID int64, bidderID int64, amount float64) (*domain.Bid, error) {
	err := s.lockManager.AcquireLock(ctx, auctionID, 5*time.Second)
	if err != nil {
		return nil, err
	}

	defer func(lockManager domain.LockManager, ctx context.Context, auctionID int64) {
		if err := lockManager.ReleaseLock(ctx, auctionID); err != nil {
			s.logger.Error("failed to release lock",
				zap.Int64("auction_id", auctionID),
				zap.Error(err),
			)
		}
	}(s.lockManager, ctx, auctionID)

	snapshot, err := s.snapShotRepo.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	if snapshot == nil {
		return nil, domain.ErrAuctionNotFound
	}

	if !snapshot.IsActive() {
		return nil, domain.ErrAuctionNotActive
	}

	highestBid, err := s.bidRepo.GetHighestBid(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	if (highestBid > 0 && amount <= highestBid) || (highestBid == 0 && amount <= snapshot.StartPrice) {
		if err := s.publisher.PublishBidRejected(ctx, &domain.Bid{
			AuctionID: auctionID,
			BotID:     bidderID,
			Amount:    amount,
		}, "bid too low"); err != nil {
			s.logger.Error("failed to publish bid.rejected event",
				zap.Int64("auction_id", auctionID),
				zap.Int64("bidder_id", bidderID),
				zap.Float64("amount", amount),
				zap.Error(err),
			)
		}
		return nil, domain.ErrBidTooLow
	}

	bid := &domain.Bid{
		AuctionID: auctionID,
		BotID:     bidderID,
		Amount:    amount,
		Status:    domain.StatusAccepted,
	}

	if err := s.bidRepo.Create(ctx, bid); err != nil {
		return nil, err
	}

	if err := s.publisher.PublishBidPlaced(ctx, bid); err != nil {
		s.logger.Error("failed to publish bid.placed event",
			zap.Int64("auction_id", auctionID),
			zap.Int64("bidder_id", bidderID),
			zap.Float64("amount", amount),
			zap.Error(err),
		)
	}

	return bid, nil
}
