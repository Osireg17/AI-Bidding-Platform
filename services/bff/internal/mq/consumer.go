package mq

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

const bffQueueName = "bff.q"

type BFFEventConsumer struct {
	*messaging.BaseConsumer
	store       domain.StateStore
	broadcaster domain.EventBroadcaster
	logger      *zap.Logger
}

func NewBFFEventConsumer(url string, store domain.StateStore, broadcaster domain.EventBroadcaster, logger *zap.Logger) (*BFFEventConsumer, error) {
	c := &BFFEventConsumer{store: store, broadcaster: broadcaster, logger: logger}

	cfg := messaging.ConsumerConfig{
		QueueName: bffQueueName,
		RoutingKeys: []string{
			events.RoutingKeyAuctionCreated,
			events.RoutingKeyAuctionEndingSoon,
			events.RoutingKeyAuctionEnded,
			events.RoutingKeyBidPlaced,
		},
		ServiceName: "bff",
	}

	base, err := messaging.NewBaseConsumer(url, cfg, c, logger)
	if err != nil {
		return nil, err
	}
	c.BaseConsumer = base
	return c, nil
}

// handleDelivery delegates to BaseConsumer so tests in package mq can reach it.
func (c *BFFEventConsumer) handleDelivery(ctx context.Context, msg messaging.Delivery) error {
	return c.BaseConsumer.HandleDelivery(ctx, msg)
}

func (c *BFFEventConsumer) Dispatch(ctx context.Context, envelope events.Envelope) error {
	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		return c.handleAuctionCreated(envelope)
	case events.RoutingKeyAuctionEndingSoon:
		return c.handleAuctionEndingSoon(envelope)
	case events.RoutingKeyAuctionEnded:
		return c.handleAuctionEnded(envelope)
	case events.RoutingKeyBidPlaced:
		return c.handleBidPlaced(envelope)
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
		return nil
	}
}

func (c *BFFEventConsumer) handleAuctionCreated(envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionCreatedPayload: %v", messaging.ErrNonRetryable, err)
	}
	c.store.ApplyAuctionCreated(payload)
	c.broadcaster.Broadcast(events.RoutingKeyAuctionCreated, payload)
	return nil
}

func (c *BFFEventConsumer) handleAuctionEndingSoon(envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndingSoonPayload: %v", messaging.ErrNonRetryable, err)
	}
	c.store.ApplyAuctionEndingSoon(payload)
	c.broadcaster.Broadcast(events.RoutingKeyAuctionEndingSoon, payload)
	return nil
}

func (c *BFFEventConsumer) handleAuctionEnded(envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", messaging.ErrNonRetryable, err)
	}
	c.store.ApplyAuctionEnded(payload)
	c.broadcaster.Broadcast(events.RoutingKeyAuctionEnded, payload)
	return nil
}

func (c *BFFEventConsumer) handleBidPlaced(envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.BidPlacedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal BidPlacedPayload: %v", messaging.ErrNonRetryable, err)
	}
	c.store.ApplyBidPlaced(payload)
	c.broadcaster.Broadcast(events.RoutingKeyBidPlaced, payload)
	return nil
}
