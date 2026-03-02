package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const botServiceQueueName = "bot.q"

var errNonRetryable = errors.New("non-retryable consumer error")

type BotEventConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	bots    []*agent.BotAgent
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

func (d amqpDelivery) Body() []byte        { return d.Delivery.Body }
func (d amqpDelivery) RoutingKey() string  { return d.Delivery.RoutingKey }
func (d amqpDelivery) DeliveryTag() uint64 { return d.Delivery.DeliveryTag }

func NewBotEventConsumer(url string, bots []*agent.BotAgent, logger *zap.Logger) (*BotEventConsumer, error) {
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
		events.ExchangeName,
		events.ExchangeKind,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare exchange %s: %w", events.ExchangeName, err)
	}

	_, err = ch.QueueDeclare(
		botServiceQueueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare queue %s: %w", botServiceQueueName, err)
	}

	routingKeys := []string{
		events.RoutingKeyAuctionCreated,
		events.RoutingKeyAuctionEndingSoon,
		events.RoutingKeyAuctionEnded,
		events.RoutingKeyBidPlaced,
	}
	for _, key := range routingKeys {
		err = ch.QueueBind(
			botServiceQueueName,
			key,
			events.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("failed to bind queue to routing key %s: %w", key, err)
		}
	}

	logger.Info("RabbitMQ consumer initialized",
		zap.String("exchange", events.ExchangeName),
		zap.Strings("routing_keys", routingKeys),
	)
	return &BotEventConsumer{conn: conn, channel: ch, bots: bots, logger: logger}, nil
}

func (c *BotEventConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		botServiceQueueName,
		"",    // consumer tag
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	c.logger.Info("bot event consumer started")
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("message channel closed")
				return errors.New("bot event consumer message channel closed")
			}
			if err := c.handleDelivery(ctx, amqpDelivery{Delivery: msg}); err != nil {
				return err
			}
		case <-ctx.Done():
			c.logger.Info("bot event consumer shutting down")
			return nil
		}
	}
}

func (c *BotEventConsumer) handleDelivery(ctx context.Context, msg delivery) error {
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

	var handleErr error
	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		handleErr = c.handleAuctionCreated(ctx, envelope)
	case events.RoutingKeyAuctionEndingSoon:
		handleErr = c.handleAuctionEndingSoon(ctx, envelope)
	case events.RoutingKeyAuctionEnded:
		handleErr = c.handleAuctionEnded(ctx, envelope)
	case events.RoutingKeyBidPlaced:
		handleErr = c.handleBidPlaced(ctx, envelope)
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

// fanOut calls Evaluate on each bot in parallel and waits for all to finish.
// Returns a non-nil error only if every bot fails (to avoid requeueing on partial failure).
func (c *BotEventConsumer) fanOut(ctx context.Context, bots []*agent.BotAgent, ac agent.AuctionContext) error {
	var wg sync.WaitGroup
	for _, b := range bots {
		wg.Add(1)
		go func(bot *agent.BotAgent) {
			defer wg.Done()
			if err := bot.Evaluate(ctx, ac); err != nil {
				c.logger.Error("bot evaluation failed",
					zap.String("bot", bot.Name()),
					zap.Int64("auction_id", ac.AuctionID),
					zap.Error(err),
				)
			}
		}(b)
	}
	wg.Wait()
	return nil
}

func (c *BotEventConsumer) handleAuctionCreated(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionCreatedPayload: %v", errNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", errNonRetryable, err)
	}

	ac := agent.AuctionContext{
		AuctionID:    payload.AuctionID,
		Title:        payload.Title,
		Description:  payload.Description,
		StartPrice:   payload.StartPrice,
		HighestBid:   payload.StartPrice,
		EndTime:      endTime,
		TriggerEvent: events.RoutingKeyAuctionCreated,
	}

	// auction.created → Alice (1), Victor (3), Charlie (4)
	bots := botsWithIDs(c.bots, 1, 3, 4)
	return c.fanOut(ctx, bots, ac)
}

func (c *BotEventConsumer) handleAuctionEndingSoon(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndingSoonPayload: %v", errNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", errNonRetryable, err)
	}

	ac := agent.AuctionContext{
		AuctionID:    payload.AuctionID,
		EndTime:      endTime,
		TriggerEvent: events.RoutingKeyAuctionEndingSoon,
	}

	// auction.ending_soon → all 4 bots
	return c.fanOut(ctx, c.bots, ac)
}

func (c *BotEventConsumer) handleAuctionEnded(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", errNonRetryable, err)
	}

	c.logger.Info("auction ended",
		zap.Int64("auction_id", payload.AuctionID),
		zap.String("final_status", payload.FinalStatus),
		zap.Int64("winner_bot_id", payload.WinnerBotID),
		zap.Float64("winning_bid", payload.WinningBid),
	)
	return nil
}

func (c *BotEventConsumer) handleBidPlaced(ctx context.Context, envelope events.Envelope) error {
	payload, err := reUnmarshalPayload[events.BidPlacedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal BidPlacedPayload: %v", errNonRetryable, err)
	}

	ac := agent.AuctionContext{
		AuctionID:    payload.AuctionID,
		HighestBid:   payload.BidAmount,
		TriggerEvent: events.RoutingKeyBidPlaced,
	}

	// bid.placed → all bots except the one that placed the bid (self-loop prevention)
	var bots []*agent.BotAgent
	for _, b := range c.bots {
		if b.ID() != payload.BotID {
			bots = append(bots, b)
		}
	}
	return c.fanOut(ctx, bots, ac)
}

func (c *BotEventConsumer) Close() error {
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

// botsWithIDs returns the subset of bots whose IDs match the provided list.
func botsWithIDs(bots []*agent.BotAgent, ids ...int64) []*agent.BotAgent {
	var result []*agent.BotAgent
	for _, b := range bots {
		for _, id := range ids {
			if b.ID() == id {
				result = append(result, b)
				break
			}
		}
	}
	return result
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
