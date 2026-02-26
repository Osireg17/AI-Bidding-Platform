package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const bidServiceQueueName = "bid.q"

type AuctionEventConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	repo    domain.AuctionSnapshotRepository
	logger  *zap.Logger
}

func NewAuctionEventConsumer(url string, repo domain.AuctionSnapshotRepository, logger *zap.Logger) (*AuctionEventConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
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
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare exchange %s: %w", events.ExchangeName, err)
	}

	_, err = ch.QueueDeclare(
		bidServiceQueueName, // name
		true,                // durable
		false,               // autoDelete
		false,               // exclusive
		false,               // noWait
		nil,                 // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare queue %s: %w", bidServiceQueueName, err)
	}

	routingKeys := []string{
		events.RoutingKeyAuctionCreated,
		events.RoutingKeyAuctionEndingSoon,
		events.RoutingKeyAuctionEnded,
	}
	for _, routingKey := range routingKeys {
		err = ch.QueueBind(
			bidServiceQueueName, // queue name
			routingKey,          // routing key
			events.ExchangeName, // exchange
			false,               // noWait
			nil,                 // arguments
		)
		if err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("failed to bind queue %s to routing key %s: %w", bidServiceQueueName, routingKey, err)
		}
	}

	logger.Info("RabbitMQ consumer initialized", zap.String("exchange", events.ExchangeName), zap.Strings("routing_keys", routingKeys))
	return &AuctionEventConsumer{conn: conn, channel: ch, repo: repo, logger: logger}, nil
}

func (c *AuctionEventConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		bidServiceQueueName, // queue
		"",                  // consumer
		false,               // autoAck
		false,               // exclusive
		false,               // noLocal
		false,               // noWait
		nil,                 // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	c.logger.Info("auction event consumer started")
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("message channel closed")
				return nil
			}
			c.handleDelivery(ctx, msg)
		case <-ctx.Done():
			c.logger.Info("auction event consumer shutting down")
			return nil
		}
	}
}

func (c *AuctionEventConsumer) handleDelivery(ctx context.Context, msg amqp.Delivery) {
	var envelope events.Envelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.logger.Error("failed to unmarshal envelope", zap.Error(err))
		if err := msg.Ack(false); err != nil {
			c.logger.Error("failed to ACK message", zap.Error(err))
		}
		return
	}

	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		if err := c.handleAuctionCreated(ctx, envelope); err != nil {
			c.logger.Error("failed to handle auction.created event", zap.Error(err))
		}
	case events.RoutingKeyAuctionEndingSoon:
		if err := c.handleAuctionEndingSoon(ctx, envelope); err != nil {
			c.logger.Error("failed to handle auction.ending_soon event", zap.Error(err))
		}
	case events.RoutingKeyAuctionEnded:
		if err := c.handleAuctionEnded(ctx, envelope); err != nil {
			c.logger.Error("failed to handle auction.ended event", zap.Error(err))
		}
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
	}

	if err := msg.Ack(false); err != nil {
		c.logger.Error("failed to ACK message", zap.Error(err))
	}
}

func (c *AuctionEventConsumer) handleAuctionCreated(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("unmarshal AuctionCreatedPayload: %w", err)
	}

	startTime, err := time.Parse(time.RFC3339, payload.StartTime)
	if err != nil {
		return fmt.Errorf("parse start time: %w", err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("parse end time: %w", err)
	}

	snapshot := &domain.AuctionSnapshot{
		AuctionID:  payload.AuctionID,
		Title:      payload.Title,
		StartPrice: payload.StartPrice,
		Status:     "active",
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
	payload, err := reUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("unmarshal AuctionEndingSoonPayload: %w", err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, "ending_soon"); err != nil {
		return fmt.Errorf("update status ending_soon: %w", err)
	}

	c.logger.Info("auction snapshot ending_soon", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

func (c *AuctionEventConsumer) handleAuctionEnded(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("unmarshal AuctionEndedPayload: %w", err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, "closed"); err != nil {
		return fmt.Errorf("update status closed: %w", err)
	}

	c.logger.Info("auction snapshot closed", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

func (c *AuctionEventConsumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.logger.Warn("error closing RabbitMQ channel", zap.Error(err))
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("error closing RabbitMQ connection: %w", err)
		}
	}
	return nil
}

// reUnmarshalPayload is a helper that re-marshals envelope.Payload (a map[string]any
// after JSON decode) into the target struct T.
// Usage: payload, err := reUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
func reUnmarshalPayload[T any](raw any) (T, error) {
	var result T
	b, err := json.Marshal(raw)
	if err != nil {
		return result, fmt.Errorf("re-marshal payload: %w", err)
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return result, fmt.Errorf("unmarshal payload: %w", err)
	}
	return result, nil
}
