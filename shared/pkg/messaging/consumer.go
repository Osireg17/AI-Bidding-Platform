package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ErrNonRetryable signals that a message should be acked and dropped rather than requeued.
var ErrNonRetryable = errors.New("non-retryable consumer error")

// Delivery abstracts an AMQP delivery for testability.
type Delivery interface {
	Body() []byte
	RoutingKey() string
	DeliveryTag() uint64
	Ack(multiple bool) error
	Nack(multiple, requeue bool) error
}

// AmqpDelivery wraps amqp.Delivery to satisfy Delivery.
type AmqpDelivery struct{ amqp.Delivery }

func (d AmqpDelivery) Body() []byte        { return d.Delivery.Body }
func (d AmqpDelivery) RoutingKey() string  { return d.Delivery.RoutingKey }
func (d AmqpDelivery) DeliveryTag() uint64 { return d.Delivery.DeliveryTag }

// Dispatcher is implemented by each service consumer to handle a decoded envelope.
// Return ErrNonRetryable (or wrap it) to ack-and-drop. Return any other error to nack-and-requeue.
type Dispatcher interface {
	Dispatch(ctx context.Context, envelope events.Envelope) error
}

// ConsumerConfig carries the queue topology for a BaseConsumer.
type ConsumerConfig struct {
	QueueName   string
	RoutingKeys []string
	ServiceName string // used in log messages only
}

// BaseConsumer owns the AMQP connection, reconnect loop, and ack/nack logic.
// Embed it in a service-specific consumer struct and supply a Dispatcher.
type BaseConsumer struct {
	url        string
	cfg        ConsumerConfig
	conn       *amqp.Connection
	channel    *amqp.Channel
	dispatcher Dispatcher
	logger     *zap.Logger
}

// NewBaseConsumerForTest builds a BaseConsumer with no live connection — for unit tests only.
func NewBaseConsumerForTest(cfg ConsumerConfig, dispatcher Dispatcher, logger *zap.Logger) *BaseConsumer {
	return &BaseConsumer{cfg: cfg, dispatcher: dispatcher, logger: logger}
}

// NewBaseConsumer dials RabbitMQ and declares the queue topology.
func NewBaseConsumer(url string, cfg ConsumerConfig, dispatcher Dispatcher, logger *zap.Logger) (*BaseConsumer, error) {
	c := &BaseConsumer{url: url, cfg: cfg, dispatcher: dispatcher, logger: logger}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *BaseConsumer) connect() error {
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

	if _, err = ch.QueueDeclare(c.cfg.QueueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to declare queue %s: %w", c.cfg.QueueName, err)
	}

	for _, key := range c.cfg.RoutingKeys {
		if err = ch.QueueBind(c.cfg.QueueName, key, events.ExchangeName, false, nil); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return fmt.Errorf("failed to bind queue %s to routing key %s: %w", c.cfg.QueueName, key, err)
		}
	}

	c.conn = conn
	c.channel = ch
	c.logger.Info("RabbitMQ consumer initialized",
		zap.String("service", c.cfg.ServiceName),
		zap.String("exchange", events.ExchangeName),
		zap.Strings("routing_keys", c.cfg.RoutingKeys),
	)
	return nil
}

// Start runs the consume loop until ctx is cancelled.
func (c *BaseConsumer) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		msgs, err := c.channel.Consume(c.cfg.QueueName, "", false, false, false, false, nil)
		if err != nil {
			c.logger.Warn("failed to start consuming, reconnecting", zap.Error(err))
			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				return reconnErr
			}
			continue
		}

		c.logger.Info("consumer started", zap.String("service", c.cfg.ServiceName))
	loop:
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("message channel closed, reconnecting")
					break loop
				}
				if err := c.HandleDelivery(ctx, AmqpDelivery{Delivery: msg}); err != nil {
					c.logger.Error("failed to handle delivery, reconnecting", zap.Error(err))
					break loop
				}
			case <-ctx.Done():
				c.logger.Info("consumer shutting down", zap.String("service", c.cfg.ServiceName))
				return nil
			}
		}

		if reconnErr := c.reconnect(ctx); reconnErr != nil {
			return reconnErr
		}
	}
}

func (c *BaseConsumer) reconnect(ctx context.Context) error {
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

func (c *BaseConsumer) HandleDelivery(ctx context.Context, msg Delivery) error {
	var envelope events.Envelope
	if err := json.Unmarshal(msg.Body(), &envelope); err != nil {
		c.logger.Error("failed to unmarshal envelope",
			zap.Error(err),
			zap.String("routing_key", msg.RoutingKey()),
		)
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack malformed envelope: %w", err)
		}
		return nil
	}

	handleErr := c.dispatcher.Dispatch(ctx, envelope)

	if handleErr == nil {
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("ack message: %w", err)
		}
		return nil
	}

	if errors.Is(handleErr, ErrNonRetryable) {
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

// Close shuts down the channel and connection.
func (c *BaseConsumer) Close() error {
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

// ReUnmarshalPayload decodes an envelope payload (stored as any after JSON round-trip) into T.
func ReUnmarshalPayload[T any](raw any) (T, error) {
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
