package mq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"go.uber.org/zap"
)

const botServiceQueueName = "bot.q"

const (
	maxBotAttempts    = 3
	botRetryBaseDelay = 200 * time.Millisecond
)

var errAllBotsFailed = errors.New("all bots failed evaluation")

// BotEvaluator is the interface satisfied by *agent.BotAgent.
type BotEvaluator interface {
	ID() int64
	Name() string
	Evaluate(ctx context.Context, ac agent.AuctionContext) error
}

type BotEventConsumer struct {
	*messaging.BaseConsumer
	bots   []BotEvaluator
	logger *zap.Logger
}

func NewBotEventConsumer(url string, bots []BotEvaluator, logger *zap.Logger) (*BotEventConsumer, error) {
	c := &BotEventConsumer{bots: bots, logger: logger}

	cfg := messaging.ConsumerConfig{
		QueueName: botServiceQueueName,
		RoutingKeys: []string{
			events.RoutingKeyAuctionCreated,
			events.RoutingKeyAuctionEndingSoon,
			events.RoutingKeyAuctionEnded,
			events.RoutingKeyBidPlaced,
		},
		ServiceName: "bot",
	}

	base, err := messaging.NewBaseConsumer(url, cfg, c, logger)
	if err != nil {
		return nil, err
	}
	c.BaseConsumer = base
	return c, nil
}

func (c *BotEventConsumer) Dispatch(ctx context.Context, envelope events.Envelope) error {
	switch envelope.EventType {
	case events.RoutingKeyAuctionCreated:
		return c.handleAuctionCreated(ctx, envelope)
	case events.RoutingKeyAuctionEndingSoon:
		return c.handleAuctionEndingSoon(ctx, envelope)
	case events.RoutingKeyAuctionEnded:
		return c.handleAuctionEnded(envelope)
	case events.RoutingKeyBidPlaced:
		return c.handleBidPlaced(ctx, envelope)
	default:
		c.logger.Warn("unknown event type", zap.String("event_type", envelope.EventType))
		return nil
	}
}

func (c *BotEventConsumer) handleAuctionCreated(ctx context.Context, envelope events.Envelope) error {
	payload, err := messaging.ReUnmarshalPayload[events.AuctionCreatedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionCreatedPayload: %v", messaging.ErrNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", messaging.ErrNonRetryable, err)
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
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndingSoonPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndingSoonPayload: %v", messaging.ErrNonRetryable, err)
	}

	endTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		return fmt.Errorf("%w: parse end time: %v", messaging.ErrNonRetryable, err)
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
	payload, err := messaging.ReUnmarshalPayload[events.AuctionEndedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal AuctionEndedPayload: %v", messaging.ErrNonRetryable, err)
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
	payload, err := messaging.ReUnmarshalPayload[events.BidPlacedPayload](envelope.Payload)
	if err != nil {
		return fmt.Errorf("%w: unmarshal BidPlacedPayload: %v", messaging.ErrNonRetryable, err)
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

// fanOut distributes auction context evaluation across bots concurrently.
// Each bot is retried up to maxBotAttempts times with exponential backoff.
// If any bot returns ErrSpendingCapExhausted the fanOut short-circuits with a non-retryable error.
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

			if errors.Is(r.err, agent.ErrSpendingCapExhausted) {
				c.logger.Error("spending cap exhausted, dropping event",
					zap.Int64("auction_id", ac.AuctionID),
					zap.String("trigger", ac.TriggerEvent),
					zap.Error(r.err),
				)
				go func(rem int) {
					for i := 0; i < rem; i++ {
						<-results
					}
				}(remaining)
				return fmt.Errorf("%w: %w", messaging.ErrNonRetryable, r.err)
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

// handleDelivery delegates to BaseConsumer so tests in package mq can reach it.
func (c *BotEventConsumer) handleDelivery(ctx context.Context, msg messaging.Delivery) error {
	return c.BaseConsumer.HandleDelivery(ctx, msg)
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
