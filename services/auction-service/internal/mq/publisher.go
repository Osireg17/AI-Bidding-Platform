package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// === CONTEXT ===
// Purpose: RabbitMQ implementation of domain.EventPublisher.
// Connects to RabbitMQ, declares the topic exchange, and publishes event envelopes.
// Reference: shared/events/envelope.go for the envelope format.
//
// === DEPENDENCIES ===
// amqp091-go — RabbitMQ client library
// shared/events — envelope constructor and payload types
// zap — structured logging
//
// === DATA / STATE ===
// RabbitMQPublisher holds a connection, channel, and logger.
// Created once at startup via NewRabbitMQPublisher. Closed at shutdown via Close.

// RabbitMQPublisher implements domain.EventPublisher using RabbitMQ.
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *zap.Logger
}

// === BEHAVIOR: NewRabbitMQPublisher ===
// Input: RabbitMQ URL string, *zap.Logger
// Output: *RabbitMQPublisher or error
// Logic:
//   DIAL RabbitMQ connection
//   OPEN a channel
//   DECLARE the topic exchange (auction.events)
//   RETURN publisher

// NewRabbitMQPublisher connects to RabbitMQ and declares the auction events exchange.
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

// === BEHAVIOR: publish (private helper) ===
// Input: context, routing key, event envelope
// Output: error if marshalling or publishing fails
// Logic:
//   MARSHAL envelope to JSON
//   PUBLISH to exchange with routing key
//   LOG success or failure

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

// PublishAuctionCreated publishes an auction.created event.
func (p *RabbitMQPublisher) PublishAuctionCreated(ctx context.Context, auction *domain.Auction) error {
	payload := events.AuctionCreatedPayload{
		AuctionID:   auction.ID,
		Title:       auction.Title,
		Description: auction.Description,
		StartPrice:  auction.StartPrice,
		StartTime:   auction.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		EndTime:     auction.EndTime.Format("2006-01-02T15:04:05Z07:00"),
	}
	envelope := events.NewEnvelope(events.RoutingKeyAuctionCreated, events.AuctionEventVersion, "", payload)
	return p.publish(ctx, events.RoutingKeyAuctionCreated, envelope)
}

// PublishAuctionEndingSoon publishes an auction.ending_soon event.
func (p *RabbitMQPublisher) PublishAuctionEndingSoon(ctx context.Context, auction *domain.Auction) error {
	payload := events.AuctionEndingSoonPayload{
		AuctionID: auction.ID,
		EndTime:   auction.EndTime.Format("2006-01-02T15:04:05Z07:00"),
	}
	envelope := events.NewEnvelope(events.RoutingKeyAuctionEndingSoon, events.AuctionEventVersion, "", payload)
	return p.publish(ctx, events.RoutingKeyAuctionEndingSoon, envelope)
}

// PublishAuctionEnded publishes an auction.ended event.
func (p *RabbitMQPublisher) PublishAuctionEnded(ctx context.Context, auction *domain.Auction) error {
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
	return p.publish(ctx, events.RoutingKeyAuctionEnded, envelope)
}

// Close shuts down the RabbitMQ channel and connection.
func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			p.logger.Warn("error closing RabbitMQ channel", zap.Error(err))
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return fmt.Errorf("error closing RabbitMQ connection: %w", err)
		}
	}
	return nil
}
