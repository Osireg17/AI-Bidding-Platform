package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const bidServiceQueueName = "bid.q"

var errNonRetryable = errors.New("non-retryable consumer error")

type AuctionEventConsumer struct {
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	repo    domain.AuctionSnapshotRepository
	logger  *zap.Logger
}

type delivery interface {
	Body() []byte
	RoutingKey() string
	DeliveryTag() uint64
	Ack(multiple bool) error
	Nack(multiple, requeue bool) error
}

type amqpDelivery struct {
	amqp.Delivery
}

func (d amqpDelivery) Body() []byte {
	return d.Delivery.Body
}

func (d amqpDelivery) RoutingKey() string {
	return d.Delivery.RoutingKey
}

func (d amqpDelivery) DeliveryTag() uint64 {
	return d.Delivery.DeliveryTag
}

func NewAuctionEventConsumer(url string, repo domain.AuctionSnapshotRepository, logger *zap.Logger) (*AuctionEventConsumer, error) {
	c := &AuctionEventConsumer{url: url, repo: repo, logger: logger}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *AuctionEventConsumer) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	if err = ch.ExchangeDeclare(events.ExchangeName, events.ExchangeKind, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to declare exchange %s: %w", events.ExchangeName, err)
	}

	if _, err = ch.QueueDeclare(bidServiceQueueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to declare queue %s: %w", bidServiceQueueName, err)
	}

	routingKeys := []string{
		events.RoutingKeyAuctionCreated,
		events.RoutingKeyAuctionEndingSoon,
		events.RoutingKeyAuctionEnded,
	}
	for _, routingKey := range routingKeys {
		if err = ch.QueueBind(bidServiceQueueName, routingKey, events.ExchangeName, false, nil); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return fmt.Errorf("failed to bind queue %s to routing key %s: %w", bidServiceQueueName, routingKey, err)
		}
	}

	c.conn = conn
	c.channel = ch
	c.logger.Info("RabbitMQ consumer initialized", zap.String("exchange", events.ExchangeName), zap.Strings("routing_keys", routingKeys))
	return nil
}

func (c *AuctionEventConsumer) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		msgs, err := c.channel.Consume(bidServiceQueueName, "", false, false, false, false, nil)
		if err != nil {
			c.logger.Warn("failed to start consuming, reconnecting", zap.Error(err))
			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				return reconnErr
			}
			continue
		}

		c.logger.Info("auction event consumer started")
	loop:
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("message channel closed, reconnecting")
					break loop
				}
				if err := c.handleDelivery(ctx, amqpDelivery{Delivery: msg}); err != nil {
					c.logger.Error("failed to handle delivery, reconnecting", zap.Error(err))
					break loop
				}
			case <-ctx.Done():
				c.logger.Info("auction event consumer shutting down")
				return nil
			}
		}

		if reconnErr := c.reconnect(ctx); reconnErr != nil {
			return reconnErr
		}
	}
}

func (c *AuctionEventConsumer) reconnect(ctx context.Context) error {
	_ = c.Close()
	for {
		if ctx.Err() != nil {
			return nil
		}
		c.logger.Info("attempting to reconnect to RabbitMQ")
		if err := c.connect(); err != nil {
			c.logger.Warn("reconnect failed, retrying in 5s", zap.Error(err))
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return nil
			}
			continue
		}
		c.logger.Info("reconnected to RabbitMQ")
		return nil
	}
}

func (c *AuctionEventConsumer) handleDelivery(ctx context.Context, msg delivery) error {
	var envelope events.Envelope
	if err := json.Unmarshal(msg.Body(), &envelope); err != nil {
		c.logger.Error("failed to unmarshal envelope",
			zap.Error(err),
			zap.String("routing_key", msg.RoutingKey()),
			zap.Uint64("delivery_tag", msg.DeliveryTag()),
			zap.Int("body_size_bytes", len(msg.Body())),
		)
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack malformed envelope: %w", err)
		}
		return nil
	}

	var handleErr error
	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		handleErr = c.handleAuctionCreated(ctx, envelope)
	case events.RoutingKeyAuctionEndingSoon:
		handleErr = c.handleAuctionEndingSoon(ctx, envelope)
	case events.RoutingKeyAuctionEnded:
		handleErr = c.handleAuctionEnded(ctx, envelope)
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack unknown event type: %w", err)
		}
		return nil
	}

	if handleErr == nil {
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack message: %w", err)
		}
		return nil
	}

	if errors.Is(handleErr, errNonRetryable) {
		c.logger.Warn("dropping non-retryable event", zap.Error(handleErr))
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack non-retryable error: %w", err)
		}
		return nil
	}

	c.logger.Error("retryable event handling failure, requeueing", zap.Error(handleErr))
	if err := msg.Nack(false, true); err != nil {
		return fmt.Errorf("nack retryable error: %w", err)
	}
	return nil
}

func (c *AuctionEventConsumer) handleAuctionCreated(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionCreatedPayload: %v", errNonRetryable, err)
	}

	startTime, err := time.Parse(time.RFC3339, payload.StartTime)
	if err != nil {
		return fmt.Errorf("%w: parse start time: %v", errNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", errNonRetryable, err)
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
	payload, err := reUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndingSoonPayload: %v", errNonRetryable, err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, domain.AuctionStatusEndingSoon); err != nil {
		return fmt.Errorf("update status ending_soon: %w", err)
	}

	c.logger.Info("auction snapshot ending_soon", zap.Int64("auction_id", payload.AuctionID))
	return nil
}

func (c *AuctionEventConsumer) handleAuctionEnded(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", errNonRetryable, err)
	}

	if err := c.repo.UpdateStatus(ctx, payload.AuctionID, domain.AuctionStatusClosed); err != nil {
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
