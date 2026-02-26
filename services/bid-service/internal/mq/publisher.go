package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *zap.Logger
}

func NewRabbitMQPublisher(url string, logger *zap.Logger) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		events.ExchangeName, // name
		events.ExchangeKind, // type (topic)
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange %s: %w", events.ExchangeName, err)
	}

	logger.Info("RabbitMQ publisher initialized", zap.String("exchange", events.ExchangeName))
	return &RabbitMQPublisher{conn: conn, channel: ch, logger: logger}, nil
}

func (p *RabbitMQPublisher) publish(ctx context.Context, routingKey string, envelope events.Envelope) error {
	body, err := json.Marshal(envelope)

	if err != nil {
		return fmt.Errorf("failed to marshal event %s: %w", envelope.EventType, err)
	}
	err = p.channel.PublishWithContext(ctx,
		events.ExchangeName, // exchange
		routingKey,          // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event %s: %w", routingKey, err)
	}

	p.logger.Info("event published",
		zap.String("routing_key", routingKey),
		zap.String("event_id", envelope.EventID),
	)
	return nil
}

func (p *RabbitMQPublisher) PublishBidPlaced(ctx context.Context, bid *domain.Bid) error {
	payload := events.BidPlacedPayload{
		AuctionID: bid.AuctionID,
		BotID:     bid.BotID,
		BidAmount: bid.Amount,
		BidID:     bid.ID,
		Timestamp: bid.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // RFC 3339
	}

	envelope := events.NewEnvelope(
		events.RoutingKeyBidPlaced, events.BidEventVersion, "", payload)

	return p.publish(ctx, events.RoutingKeyBidPlaced, envelope)
}
