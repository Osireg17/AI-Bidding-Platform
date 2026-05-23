package mq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/agent"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeBot struct {
	id          int64
	name        string
	mu          sync.Mutex
	calls       []agent.AuctionContext
	errToReturn error
	failTimes   int // -1 = always fail; 0 = always succeed; N = fail first N calls
}

func (f *fakeBot) ID() int64    { return f.id }
func (f *fakeBot) Name() string { return f.name }

func (f *fakeBot) Evaluate(_ context.Context, ac agent.AuctionContext) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, ac)
	if f.failTimes == -1 {
		return f.errToReturn
	}
	if f.failTimes > 0 {
		f.failTimes--
		return f.errToReturn
	}
	return nil
}

func (f *fakeBot) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeBot) lastCall() agent.AuctionContext {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[len(f.calls)-1]
}

type fakeDelivery struct {
	body        []byte
	routingKey  string
	deliveryTag uint64
	ackCalls    int
	nackCalls   int
	nackRequeue bool
}

func (d *fakeDelivery) Body() []byte        { return d.body }
func (d *fakeDelivery) RoutingKey() string  { return d.routingKey }
func (d *fakeDelivery) DeliveryTag() uint64 { return d.deliveryTag }
func (d *fakeDelivery) Ack(_ bool) error    { d.ackCalls++; return nil }
func (d *fakeDelivery) Nack(_ bool, requeue bool) error {
	d.nackCalls++
	d.nackRequeue = requeue
	return nil
}

func newTestConsumer(bots []BotEvaluator) *BotEventConsumer {
	c := &BotEventConsumer{bots: bots, logger: zap.NewNop()}
	c.BaseConsumer = messaging.NewBaseConsumerForTest(messaging.ConsumerConfig{}, c, zap.NewNop())
	return c
}

func allFakeBots() []*fakeBot {
	return []*fakeBot{
		{id: 1, name: "Aggressive Alice"},
		{id: 2, name: "Sniper Steve"},
		{id: 3, name: "Value Victor"},
		{id: 4, name: "Chaos Charlie"},
	}
}

func toBotEvaluators(fbs []*fakeBot) []BotEvaluator {
	out := make([]BotEvaluator, len(fbs))
	for i, b := range fbs {
		out[i] = b
	}
	return out
}

func makeDelivery(t *testing.T, eventType string, payload any) *fakeDelivery {
	t.Helper()
	envelope := events.NewEnvelope(eventType, events.AuctionEventVersion, "", payload)
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	return &fakeDelivery{body: body, routingKey: eventType}
}

func makeAuctionCreatedPayload() events.AuctionCreatedPayload {
	now := time.Now().UTC()
	return events.AuctionCreatedPayload{
		AuctionID:   10,
		Title:       "Vintage Watch",
		Description: "Rare 1960s piece",
		StartPrice:  200.0,
		StartTime:   now.Format(time.RFC3339),
		EndTime:     now.Add(2 * time.Hour).Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// handleDelivery — routing
// ---------------------------------------------------------------------------

func TestHandleDelivery_MalformedJSON_IsAckedAndIgnored(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	d := &fakeDelivery{body: []byte("not-json"), routingKey: "auction.created"}

	err := consumer.handleDelivery(context.Background(), d)
	require.NoError(t, err)

	assert.Equal(t, 1, d.ackCalls, "malformed message must be acked (drop)")
	assert.Equal(t, 0, d.nackCalls)
	for _, b := range bots {
		assert.Equal(t, 0, b.callCount(), "no bot should be called on malformed message")
	}
}

func TestHandleDelivery_UnknownEventType_IsAckedAndIgnored(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))
	d := makeDelivery(t, "some.unknown.event", map[string]any{"foo": "bar"})

	err := consumer.handleDelivery(context.Background(), d)
	require.NoError(t, err)

	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	for _, b := range bots {
		assert.Equal(t, 0, b.callCount())
	}
}

// ---------------------------------------------------------------------------
// handleDelivery — auction.created
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionCreated_CallsAliceVictorCharlie(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, makeAuctionCreatedPayload())

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	assert.Equal(t, 1, bots[0].callCount(), "Alice should be called")
	assert.Equal(t, 0, bots[1].callCount(), "Steve should NOT be called")
	assert.Equal(t, 1, bots[2].callCount(), "Victor should be called")
	assert.Equal(t, 1, bots[3].callCount(), "Charlie should be called")
}

func TestHandleDelivery_AuctionCreated_ContextPassedCorrectly(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))
	payload := makeAuctionCreatedPayload()
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	ac := bots[0].lastCall() // Alice
	assert.Equal(t, payload.AuctionID, ac.AuctionID)
	assert.Equal(t, payload.Title, ac.Title)
	assert.Equal(t, payload.Description, ac.Description)
	assert.Equal(t, payload.StartPrice, ac.StartPrice)
	assert.Equal(t, payload.StartPrice, ac.HighestBid, "HighestBid must equal StartPrice on creation")
	assert.Equal(t, events.RoutingKeyAuctionCreated, ac.TriggerEvent)
}

func TestHandleDelivery_AuctionCreated_InvalidEndTime_IsNonRetryable(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	payload := events.AuctionCreatedPayload{
		AuctionID:  10,
		Title:      "Watch",
		StartPrice: 50,
		StartTime:  time.Now().Format(time.RFC3339),
		EndTime:    "not-a-time",
	}
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	// Non-retryable → acked and dropped
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
}

// ---------------------------------------------------------------------------
// handleDelivery — auction.ending_soon
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionEndingSoon_CallsAllBots(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	payload := events.AuctionEndingSoonPayload{
		AuctionID: 20,
		EndTime:   time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	d := makeDelivery(t, events.RoutingKeyAuctionEndingSoon, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	for _, b := range bots {
		assert.Equal(t, 1, b.callCount(), "all bots must be called for ending_soon")
	}
}

func TestHandleDelivery_AuctionEndingSoon_ContextPassedCorrectly(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	endTime := time.Now().UTC().Add(5 * time.Minute)
	payload := events.AuctionEndingSoonPayload{
		AuctionID: 20,
		EndTime:   endTime.Format(time.RFC3339),
	}
	d := makeDelivery(t, events.RoutingKeyAuctionEndingSoon, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	ac := bots[0].lastCall()
	assert.Equal(t, int64(20), ac.AuctionID)
	assert.Equal(t, events.RoutingKeyAuctionEndingSoon, ac.TriggerEvent)
	assert.WithinDuration(t, endTime, ac.EndTime, time.Second)
}

// ---------------------------------------------------------------------------
// handleDelivery — auction.ended
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionEnded_IsLoggedAndAcked(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	payload := events.AuctionEndedPayload{
		AuctionID:   30,
		FinalStatus: "sold",
		WinnerBotID: 1,
		WinningBid:  300.0,
		TotalBids:   7,
	}
	d := makeDelivery(t, events.RoutingKeyAuctionEnded, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	// No Evaluate calls expected — auction.ended is informational only.
	for _, b := range bots {
		assert.Equal(t, 0, b.callCount())
	}
}

// ---------------------------------------------------------------------------
// handleDelivery — bid.placed
// ---------------------------------------------------------------------------

func TestHandleDelivery_BidPlaced_ExcludesBiddingBot(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	// Steve (id=2) placed the bid — Steve must NOT receive the event.
	payload := events.BidPlacedPayload{AuctionID: 40, BotID: 2, BidAmount: 120.0}
	d := makeDelivery(t, events.RoutingKeyBidPlaced, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	assert.Equal(t, 1, bots[0].callCount(), "Alice should be called")
	assert.Equal(t, 0, bots[1].callCount(), "Steve (bidder) must be excluded")
	assert.Equal(t, 1, bots[2].callCount(), "Victor should be called")
	assert.Equal(t, 1, bots[3].callCount(), "Charlie should be called")
}

func TestHandleDelivery_BidPlaced_ContextPassedCorrectly(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	payload := events.BidPlacedPayload{AuctionID: 40, BotID: 2, BidAmount: 120.0}
	d := makeDelivery(t, events.RoutingKeyBidPlaced, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	ac := bots[0].lastCall() // Alice
	assert.Equal(t, int64(40), ac.AuctionID)
	assert.Equal(t, 120.0, ac.HighestBid)
	assert.Equal(t, events.RoutingKeyBidPlaced, ac.TriggerEvent)
}

func TestHandleDelivery_BidPlaced_AllBotsIncludedWhenBotIDIsZero(t *testing.T) {
	bots := allFakeBots()
	consumer := newTestConsumer(toBotEvaluators(bots))

	// BotID 0 matches no known bot — all should receive the event.
	payload := events.BidPlacedPayload{AuctionID: 50, BotID: 0, BidAmount: 75.0}
	d := makeDelivery(t, events.RoutingKeyBidPlaced, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	for _, b := range bots {
		assert.Equal(t, 1, b.callCount())
	}
}

// ---------------------------------------------------------------------------
// fanOut — retry & failure propagation
// ---------------------------------------------------------------------------

func TestFanOut_TransientFailure_RetriesAndSucceeds(t *testing.T) {
	transient := &fakeBot{
		id:          1,
		name:        "Aggressive Alice",
		errToReturn: errors.New("model timeout"),
		failTimes:   maxBotAttempts - 1, // fail first 2, succeed on 3rd
	}
	consumer := newTestConsumer([]BotEvaluator{transient})

	payload := makeAuctionCreatedPayload()
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls, "message must be ACKed after eventual success")
	assert.Equal(t, 0, d.nackCalls)
	assert.Equal(t, maxBotAttempts, transient.callCount(), "bot must be called exactly maxBotAttempts times")
}

func TestFanOut_AllBotsAlwaysFail_NacksMessage(t *testing.T) {
	persistentErr := errors.New("service unavailable")
	bots := []*fakeBot{
		{id: 1, name: "Alice", errToReturn: persistentErr, failTimes: -1},
		{id: 3, name: "Victor", errToReturn: persistentErr, failTimes: -1},
		{id: 4, name: "Charlie", errToReturn: persistentErr, failTimes: -1},
	}
	consumer := newTestConsumer(toBotEvaluators(bots))

	payload := makeAuctionCreatedPayload()
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 0, d.ackCalls, "message must NOT be acked when all bots fail")
	assert.Equal(t, 1, d.nackCalls, "message must be NACKed for requeue")
	assert.True(t, d.nackRequeue, "message must be requeued")

	// Each bot should have been attempted maxBotAttempts times.
	for _, b := range bots {
		assert.Equal(t, maxBotAttempts, b.callCount(),
			"bot %s should be retried %d times", b.name, maxBotAttempts)
	}
}

func TestFanOut_PartialSuccess_Acks(t *testing.T) {
	persistentErr := errors.New("service unavailable")
	alice := &fakeBot{id: 1, name: "Alice"}                                              // always succeeds
	victor := &fakeBot{id: 3, name: "Victor", errToReturn: persistentErr, failTimes: -1} // always fails
	charlie := &fakeBot{id: 4, name: "Charlie", errToReturn: persistentErr, failTimes: -1}

	consumer := newTestConsumer([]BotEvaluator{alice, victor, charlie})

	payload := makeAuctionCreatedPayload()
	d := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))

	assert.Equal(t, 1, d.ackCalls, "partial success must still ACK")
	assert.Equal(t, 0, d.nackCalls)
	assert.Equal(t, 1, alice.callCount())
	assert.Equal(t, maxBotAttempts, victor.callCount())
	assert.Equal(t, maxBotAttempts, charlie.callCount())
}

func TestFanOut_EmptyBotList_Acks(t *testing.T) {
	consumer := newTestConsumer(nil)

	// auction.ending_soon → c.bots (empty)
	payload := events.AuctionEndingSoonPayload{
		AuctionID: 20,
		EndTime:   time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	d := makeDelivery(t, events.RoutingKeyAuctionEndingSoon, payload)

	require.NoError(t, consumer.handleDelivery(context.Background(), d))
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
}

func TestFanOut_ContextCancelledDuringBackoff_DoesNotBlock(t *testing.T) {
	slow := &fakeBot{
		id:          1,
		name:        "Alice",
		errToReturn: errors.New("slow model"),
		failTimes:   -1, // always fail so it would hit the backoff sleep
	}
	consumer := newTestConsumer([]BotEvaluator{slow})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after the first Evaluate call returns — this happens before the
	// first backoff sleep, so the goroutine should exit quickly.
	go func() {
		for slow.callCount() < 1 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	start := time.Now()
	// fanOut itself — we call it directly to avoid AMQP ACK/NACK machinery.
	ac := agent.AuctionContext{AuctionID: 99, TriggerEvent: events.RoutingKeyAuctionCreated}
	_ = consumer.fanOut(ctx, []BotEvaluator{slow}, ac)
	elapsed := time.Since(start)

	// The full backoff for 3 attempts would be 200ms + 400ms = 600ms.
	// With context cancellation after the first call it should be well under that.
	assert.Less(t, elapsed, 500*time.Millisecond,
		"fanOut must not block for full backoff when context is cancelled")
}

// ---------------------------------------------------------------------------
// botsWithIDs
// ---------------------------------------------------------------------------

func TestBotsWithIDs_ReturnsMatchingSubset(t *testing.T) {
	bots := toBotEvaluators(allFakeBots())
	result := botsWithIDs(bots, 1, 3)
	require.Len(t, result, 2)
	ids := []int64{result[0].ID(), result[1].ID()}
	assert.Contains(t, ids, int64(1))
	assert.Contains(t, ids, int64(3))
}

func TestBotsWithIDs_NoneMatch_ReturnsEmpty(t *testing.T) {
	result := botsWithIDs(toBotEvaluators(allFakeBots()), 99, 100)
	assert.Empty(t, result)
}

func TestBotsWithIDs_AllMatch(t *testing.T) {
	result := botsWithIDs(toBotEvaluators(allFakeBots()), 1, 2, 3, 4)
	assert.Len(t, result, 4)
}

func TestBotsWithIDs_EmptyInput(t *testing.T) {
	assert.Empty(t, botsWithIDs(nil, 1, 2))
}

// ---------------------------------------------------------------------------
// messaging.ReUnmarshalPayload
// ---------------------------------------------------------------------------

func TestReUnmarshalPayload_AuctionCreated(t *testing.T) {
	now := time.Now().UTC()
	raw := map[string]any{
		"auction_id":  float64(7),
		"title":       "Antique Clock",
		"description": "17th century",
		"start_price": float64(500.0),
		"start_time":  now.Format(time.RFC3339),
		"end_time":    now.Add(24 * time.Hour).Format(time.RFC3339),
	}
	result, err := messaging.ReUnmarshalPayload[events.AuctionCreatedPayload](raw)
	require.NoError(t, err)
	assert.Equal(t, int64(7), result.AuctionID)
	assert.Equal(t, "Antique Clock", result.Title)
	assert.Equal(t, 500.0, result.StartPrice)
}

func TestReUnmarshalPayload_BidPlaced(t *testing.T) {
	raw := map[string]any{
		"auction_id": float64(8),
		"bot_id":     float64(2),
		"bid_amount": float64(250.0),
	}
	result, err := messaging.ReUnmarshalPayload[events.BidPlacedPayload](raw)
	require.NoError(t, err)
	assert.Equal(t, int64(8), result.AuctionID)
	assert.Equal(t, int64(2), result.BotID)
	assert.Equal(t, 250.0, result.BidAmount)
}

func TestReUnmarshalPayload_InvalidPayload_ReturnsError(t *testing.T) {
	_, err := messaging.ReUnmarshalPayload[events.AuctionCreatedPayload](make(chan int))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Compile-time interface guard
// ---------------------------------------------------------------------------

var _ BotEvaluator = (*fakeBot)(nil)
