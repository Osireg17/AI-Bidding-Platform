package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/bidclient"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"go.uber.org/zap"
)

type AuctionService struct {
	repo         domain.AuctionRepository
	publisher    domain.EventPublisher
	bidClient    *bidclient.BidServiceClient
	auctionAgent *agent.AuctionAgent
	logger       *zap.Logger
}

func NewAuctionService(repo domain.AuctionRepository, publisher domain.EventPublisher, bidClient *bidclient.BidServiceClient, auctionAgent *agent.AuctionAgent, logger *zap.Logger) *AuctionService {
	return &AuctionService{
		repo:         repo,
		publisher:    publisher,
		bidClient:    bidClient,
		auctionAgent: auctionAgent,
		logger:       logger,
	}
}

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
	}

	s.logger.Info("auction created",
		zap.Int64("auction_id", auction.ID),
		zap.String("title", auction.Title),
		zap.Time("end_time", auction.EndTime),
	)
	return auction, nil
}

func (s *AuctionService) GetAuction(ctx context.Context, id int64) (*domain.Auction, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AuctionService) ListAuctions(ctx context.Context) ([]*domain.Auction, error) {
	return s.repo.List(ctx)
}

func (s *AuctionService) ProcessExpiredAuctions(ctx context.Context) error {
	expired, err := s.repo.FindExpiredActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to find expired auctions: %w", err)
	}

	for _, auction := range expired {
		// Query the bid-service for the winning bot and amount before closing.
		var winnerBotID int64
		var winningAmount float64
		if s.bidClient != nil {
			winnerBotID, winningAmount, err = s.bidClient.GetWinner(ctx, auction.ID)
			if err != nil {
				s.logger.Warn("failed to get winner from bid-service, closing as unsold",
					zap.Int64("auction_id", auction.ID),
					zap.Error(err),
				)
				winnerBotID = 0
				winningAmount = 0
			}
		}

		if err := auction.Close(winnerBotID, winningAmount); err != nil {
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

		finalStatus := "unsold"
		if winnerBotID > 0 {
			finalStatus = "sold"
		}
		s.logger.Info("auction closed",
			zap.Int64("auction_id", auction.ID),
			zap.String("final_status", finalStatus),
			zap.Int64("winner_bot_id", winnerBotID),
			zap.Float64("winning_amount", winningAmount),
		)

		// After closing, wait one hour then generate and create the next auction.
		if s.auctionAgent != nil {
			delay := 30 * time.Minute
			s.logger.Info("scheduling next auction", zap.Duration("delay", delay))

			select {
			case <-ctx.Done():
				s.logger.Info("context cancelled, skipping next auction creation")
				return nil
			case <-time.After(delay):
			}

			item, err := s.auctionAgent.Generate(ctx)
			if err != nil {
				s.logger.Error("failed to generate next auction item", zap.Error(err))
				continue
			}

			next, err := s.CreateAuction(ctx, item.Title, item.Description, item.StartPrice, time.Duration(item.DurationSec)*time.Second)
			if err != nil {
				s.logger.Error("failed to create next auction", zap.Error(err))
			} else {
				s.logger.Info("next auction created", zap.Int64("auction_id", next.ID), zap.String("title", next.Title))
			}
		}
	}
	return nil
}

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
