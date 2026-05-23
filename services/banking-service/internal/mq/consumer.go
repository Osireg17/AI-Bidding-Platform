package mq

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

const bankingServiceQueue = "banking.q"

// BankingService defines the subset of the banking service used by the consumer.
type BankingService interface {
	RecordWin(ctx context.Context, botID, auctionID int64, title string, winningBid float64) (float64, error)
}

type BankingEventConsumer struct {
	*messaging.BaseConsumer
	svc    BankingService
	logger *zap.Logger
}

func NewBankingEventConsumer(url string, svc BankingService, logger *zap.Logger) (*BankingEventConsumer, error) {
	c := &BankingEventConsumer{svc: svc, logger: logger}

	cfg := messaging.ConsumerConfig{
		QueueName:   bankingServiceQueue,
		RoutingKeys: []string{events.RoutingKeyAuctionEnded},
		ServiceName: "banking",
	}

	base, err := messaging.NewBaseConsumer(url, cfg, c, logger)
	if err != nil {
		return nil, err
	}
	c.BaseConsumer = base
	return c, nil
}

func (c *BankingEventConsumer) Dispatch(ctx context.Context, envelope events.Envelope) error {
	switch envelope.EventType {
	case events.RoutingKeyAuctionEnded:
		return c.handleAuctionEnded(ctx, envelope)
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
		return nil
	}
}

func (c *BankingEventConsumer) handleAuctionEnded(ctx context.Context, envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", messaging.ErrNonRetryable, err)
	}

	if payload.FinalStatus != "sold" || payload.WinnerBotID <= 0 {
		c.logger.Info("auction ended without a winner, skipping",
			zap.Int64("auction_id", payload.AuctionID),
			zap.String("final_status", payload.FinalStatus),
		)
		return nil
	}

	if _, err := c.svc.RecordWin(ctx, payload.WinnerBotID, payload.AuctionID, payload.Title, payload.WinningBid); err != nil {
		return fmt.Errorf("record win for auction %d: %w", payload.AuctionID, err)
	}

	c.logger.Info("win recorded",
		zap.Int64("auction_id", payload.AuctionID),
		zap.Int64("winner_bot_id", payload.WinnerBotID),
		zap.Float64("winning_bid", payload.WinningBid),
	)
	return nil
}
