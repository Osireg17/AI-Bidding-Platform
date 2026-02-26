package mq

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

// AuctionPublisher wraps the shared RabbitMQ publisher with auction-specific methods.
type AuctionPublisher struct {
	*messaging.RabbitMQPublisher
}

// NewAuctionPublisher creates a new auction event publisher.
func NewAuctionPublisher(url string, logger *zap.Logger) (*AuctionPublisher, error) {
	base, err := messaging.NewRabbitMQPublisher(url, logger)
	if err != nil {
		return nil, err
	}
	return &AuctionPublisher{RabbitMQPublisher: base}, nil
}

// PublishAuctionCreated publishes an auction.created event.
func (p *AuctionPublisher) PublishAuctionCreated(ctx context.Context, auction *domain.Auction) error {
	payload := events.AuctionCreatedPayload{
		AuctionID:   auction.ID,
		Title:       auction.Title,
		Description: auction.Description,
		StartPrice:  auction.StartPrice,
		StartTime:   auction.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		EndTime:     auction.EndTime.Format("2006-01-02T15:04:05Z07:00"),
	}
	envelope := events.NewEnvelope(events.RoutingKeyAuctionCreated, events.AuctionEventVersion, "", payload)
	return p.Publish(ctx, events.RoutingKeyAuctionCreated, envelope)
}

// PublishAuctionEndingSoon publishes an auction.ending_soon event.
func (p *AuctionPublisher) PublishAuctionEndingSoon(ctx context.Context, auction *domain.Auction) error {
	payload := events.AuctionEndingSoonPayload{
		AuctionID: auction.ID,
		EndTime:   auction.EndTime.Format("2006-01-02T15:04:05Z07:00"),
	}
	envelope := events.NewEnvelope(events.RoutingKeyAuctionEndingSoon, events.AuctionEventVersion, "", payload)
	return p.Publish(ctx, events.RoutingKeyAuctionEndingSoon, envelope)
}

// PublishAuctionEnded publishes an auction.ended event.
func (p *AuctionPublisher) PublishAuctionEnded(ctx context.Context, auction *domain.Auction) error {
	finalStatus := "unsold"
	if auction.WinnerBotID != 0 {
		finalStatus = "sold"
	}
	payload := events.AuctionEndedPayload{
		AuctionID:   auction.ID,
		WinnerBotID: auction.WinnerBotID,
		WinningBid:  auction.CurrentPrice,
		FinalStatus: finalStatus,
	}
	envelope := events.NewEnvelope(events.RoutingKeyAuctionEnded, events.AuctionEventVersion, "", payload)
	return p.Publish(ctx, events.RoutingKeyAuctionEnded, envelope)
}
