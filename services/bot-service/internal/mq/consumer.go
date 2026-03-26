package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const botServiceQueueName = "bot.q"

// fanOut retry policy.
const (
	maxBotAttempts    = 3
	botRetryBaseDelay = 200 * time.Millisecond
)

var (
	errNonRetryable  = errors.New("non-retryable consumer error")
	errAllBotsFailed = errors.New("all bots failed evaluation")
)

// BotEvaluator is the interface satisfied by *agent.BotAgent. It is extracted
// here so that the consumer can be unit-tested without a live Gemini model.
type BotEvaluator interface {
	ID() int64
	Name() string
	Evaluate(ctx context.Context, ac agent.AuctionContext) error
}

type BotEventConsumer struct {
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	bots    []BotEvaluator
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

func NewBotEventConsumer(url string, bots []BotEvaluator, logger *zap.Logger) (*BotEventConsumer, error) {
	c := &BotEventConsumer{url: url, bots: bots, logger: logger}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *BotEventConsumer) connect() error {
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

	if _, err = ch.QueueDeclare(botServiceQueueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to declare queue %s: %w", botServiceQueueName, err)
	}

	routingKeys := []string{
		events.RoutingKeyAuctionCreated,
		events.RoutingKeyAuctionEndingSoon,
		events.RoutingKeyAuctionEnded,
		events.RoutingKeyBidPlaced,
	}
	for _, key := range routingKeys {
		if err = ch.QueueBind(botServiceQueueName, key, events.ExchangeName, false, nil); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return fmt.Errorf("failed to bind queue to routing key %s: %w", key, err)
		}
	}

	c.conn = conn
	c.channel = ch
	c.logger.Info("RabbitMQ consumer initialized",
		zap.String("exchange", events.ExchangeName),
		zap.Strings("routing_keys", routingKeys),
	)
	return nil
}

func (c *BotEventConsumer) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		msgs, err := c.channel.Consume(botServiceQueueName, "", false, false, false, false, nil)
		if err != nil {
			c.logger.Warn("failed to start consuming, reconnecting", zap.Error(err))
			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				return reconnErr
			}
			continue
		}

		c.logger.Info("bot event consumer started")
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
				c.logger.Info("bot event consumer shutting down")
				return nil
			}
		}

		if reconnErr := c.reconnect(ctx); reconnErr != nil {
			return reconnErr
		}
	}
}

func (c *BotEventConsumer) reconnect(ctx context.Context) error {
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
		handleErr = c.handleAuctionEnded(envelope)
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

// fanOut distributes auction context evaluation across the provided bots concurrently.
// Each bot is retried up to maxBotAttempts times with exponential backoff.
//
// If any bot returns ErrSpendingCapExhausted the entire fanOut short-circuits and
// returns a non-retryable error — retrying or requeuing would just burn more quota.
func (c *BotEventConsumer) fanOut(ctx context.Context, bots []BotEvaluator, ac agent.AuctionContext) error {
	if len(bots) == 0 {
		return nil
	}

	type botResult struct {
		name    string
		success bool
		err     error
	}

	results := make(chan botResult, len(bots))

	for _, b := range bots {
		go func(bot BotEvaluator) {
			var lastErr error
			for attempt := 1; attempt <= maxBotAttempts; attempt++ {
				lastErr = bot.Evaluate(ctx, ac)
				if lastErr == nil {
					results <- botResult{name: bot.Name(), success: true}
					return
				}

				if errors.Is(lastErr, agent.ErrSpendingCapExhausted) {
					results <- botResult{name: bot.Name(), success: false, err: lastErr}
					return
				}

				c.logger.Warn("bot evaluation failed, will retry",
					zap.String("bot", bot.Name()),
					zap.Int64("auction_id", ac.AuctionID),
					zap.String("trigger", ac.TriggerEvent),
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", maxBotAttempts),
					zap.Error(lastErr),
				)

				if attempt < maxBotAttempts {
					delay := botRetryBaseDelay * (1 << (attempt - 1)) // 200ms, 400ms
					select {
					case <-time.After(delay):
					case <-ctx.Done():
						results <- botResult{name: bot.Name(), success: false, err: ctx.Err()}
						return
					}
				}
			}

			c.logger.Error("bot evaluation exhausted all attempts",
				zap.String("bot", bot.Name()),
				zap.Int64("auction_id", ac.AuctionID),
				zap.String("trigger", ac.TriggerEvent),
				zap.Int("attempts", maxBotAttempts),
				zap.Error(lastErr),
			)
			results <- botResult{name: bot.Name(), success: false, err: lastErr}
		}(b)
	}

	successCount := 0
	remaining := len(bots)
	for remaining > 0 {
		select {
		case r := <-results:
			remaining--
			if r.success {
				successCount++
			}

			// Check if this bot failed due to spending cap.
			if errors.Is(r.err, agent.ErrSpendingCapExhausted) {
				c.logger.Error("spending cap exhausted, dropping event",
					zap.Int64("auction_id", ac.AuctionID),
					zap.String("trigger", ac.TriggerEvent),
					zap.Error(r.err),
				)
				// Drain remaining results from channel to avoid goroutine leaks.
				go func(rem int) {
					for i := 0; i < rem; i++ {
						<-results
					}
				}(remaining)
				return fmt.Errorf("%w: %w", errNonRetryable, r.err)
			}

		case <-ctx.Done():
			return fmt.Errorf("fanOut cancelled: auction_id=%d trigger=%s: %w", ac.AuctionID, ac.TriggerEvent, ctx.Err())
		}
	}

	if successCount == 0 {
		return fmt.Errorf("%w: auction_id=%d trigger=%s", errAllBotsFailed, ac.AuctionID, ac.TriggerEvent)
	}
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

func (c *BotEventConsumer) handleAuctionEnded(envelope events.Envelope) error {
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
	var bots []BotEvaluator
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
func botsWithIDs(bots []BotEvaluator, ids ...int64) []BotEvaluator {
	var result []BotEvaluator
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
