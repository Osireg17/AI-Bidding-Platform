package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQPublisher provides a base RabbitMQ publisher with common functionality.
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *zap.Logger
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher and declares the exchange.
func NewRabbitMQPublisher(url string, logger *zap.Logger) (*RabbitMQPublisher, error) {
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

	logger.Info("RabbitMQ publisher initialized", zap.String("exchange", events.ExchangeName))
	return &RabbitMQPublisher{conn: conn, channel: ch, logger: logger}, nil
}

// Publish publishes an event envelope to the exchange with the given routing key.
func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, envelope events.Envelope) error {
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
