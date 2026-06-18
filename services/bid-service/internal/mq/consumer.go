package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

const bidServiceQueueName = "bid.q"

type AuctionEventConsumer struct {
	*messaging.BaseConsumer
	repo   domain.AuctionSnapshotRepository
	logger *zap.Logger
}

func NewAuctionEventConsumer(url string, repo domain.AuctionSnapshotRepository, logger *zap.Logger) (*AuctionEventConsumer, error) {
	c := &AuctionEventConsumer{repo: repo, logger: logger}

	cfg := messaging.ConsumerConfig{
		QueueName: bidServiceQueueName,
		RoutingKeys: []string{
			events.RoutingKeyAuctionCreated,
			events.RoutingKeyAuctionEndingSoon,
			events.RoutingKeyAuctionEnded,
		},
		ServiceName: "bid",
	}

	base, err := messaging.NewBaseConsumer(url, cfg, c, logger)
	if err != nil {
		return nil, err
	}
	c.BaseConsumer = base
	return c, nil
}

func (c *AuctionEventConsumer) Dispatch(ctx context.Context, envelope events.Envelope) error {
	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		return c.handleAuctionCreated(ctx, envelope)
	case events.RoutingKeyAuctionEndingSoon:
		return c.handleAuctionEndingSoon(ctx, envelope)
	case events.RoutingKeyAuctionEnded:
		return c.handleAuctionEnded(ctx, envelope)
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
		return nil
	}
}

func (c *AuctionEventConsumer) handleAuctionCreated(ctx context.Context, envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionCreatedPayload: %v", messaging.ErrNonRetryable, err)
	}

	startTime, err := time.Parse(time.RFC3339, payload.StartTime)
	if err != nil {
		return fmt.Errorf("%w: parse start time: %v", messaging.ErrNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", messaging.ErrNonRetryable, err)
	}

	snapshot := &domain.AuctionSnapshot{
		AuctionID:  payload.AuctionID,
		Title:      payload.Title,
		StartPrice: payload.StartPrice,
		Status:     domain.AuctionStatusActive,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	if err := c.repo.Upsert(ctx, snapshot); err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}

	c.logger.Info("auction snapshot created", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

func (c *AuctionEventConsumer) handleAuctionEndingSoon(ctx context.Context, envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndingSoonPayload: %v", messaging.ErrNonRetryable, err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, domain.AuctionStatusEndingSoon); err != nil {
		return fmt.Errorf("update status ending_soon: %w", err)
	}

	c.logger.Info("auction snapshot ending_soon", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

func (c *AuctionEventConsumer) handleAuctionEnded(ctx context.Context, envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", messaging.ErrNonRetryable, err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, domain.AuctionStatusClosed); err != nil {
		return fmt.Errorf("update status closed: %w", err)
	}

	c.logger.Info("auction snapshot closed", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

// handleDelivery delegates to BaseConsumer so tests in package mq can reach it.
func (c *AuctionEventConsumer) handleDelivery(ctx context.Context, msg messaging.Delivery) error {
	return c.BaseConsumer.HandleDelivery(ctx, msg)
}
