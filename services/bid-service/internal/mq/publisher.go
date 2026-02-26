package mq

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

// BidPublisher wraps the shared RabbitMQ publisher with bid-specific methods.
type BidPublisher struct {
	*messaging.RabbitMQPublisher
}

// NewBidPublisher creates a new bid event publisher.
func NewBidPublisher(url string, logger *zap.Logger) (*BidPublisher, error) {
	base, err := messaging.NewRabbitMQPublisher(url, logger)
	if err != nil {
		return nil, err
	}
	return &BidPublisher{RabbitMQPublisher: base}, nil
}

// PublishBidPlaced publishes a bid.placed event.
func (p *BidPublisher) PublishBidPlaced(ctx context.Context, bid *domain.Bid) error {
	payload := events.BidPlacedPayload{
		AuctionID: bid.AuctionID,
		BotID:     bid.BotID,
		BidAmount: bid.Amount,
		BidID:     bid.ID,
		Timestamp: bid.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // RFC 3339
	}

	envelope := events.NewEnvelope(
		events.RoutingKeyBidPlaced, events.BidEventVersion, "", payload)

	return p.Publish(ctx, events.RoutingKeyBidPlaced, envelope)
}

// PublishBidRejected publishes a bid.rejected event.
func (p *BidPublisher) PublishBidRejected(ctx context.Context, bid *domain.Bid, reason string) error {
	payload := events.BidRejectedPayload{
		AuctionID: bid.AuctionID,
		BotID:     bid.BotID,
		BidAmount: bid.Amount,
		BidID:     bid.ID,
		Reason:    reason,
		Timestamp: bid.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // RFC 3339
	}

	envelope := events.NewEnvelope(
		events.RoutingKeyBidRejected, events.BidEventVersion, "", payload)

	return p.Publish(ctx, events.RoutingKeyBidRejected, envelope)
}
