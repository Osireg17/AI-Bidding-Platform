package mq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stateStoreSpy struct {
	createdPayload    *events.AuctionCreatedPayload
	endingSoonPayload *events.AuctionEndingSoonPayload
	endedPayload      *events.AuctionEndedPayload
	bidPlacedPayload  *events.BidPlacedPayload
	operationOrderLog *[]string
}

func (s *stateStoreSpy) GetState() domain.AuctionState {
	return domain.AuctionState{}
}

func (s *stateStoreSpy) ApplyAuctionCreated(payload events.AuctionCreatedPayload) {
	s.createdPayload = &payload
	if s.operationOrderLog != nil {
		*s.operationOrderLog = append(*s.operationOrderLog, "store.auction_created")
	}
}

func (s *stateStoreSpy) ApplyAuctionEndingSoon(payload events.AuctionEndingSoonPayload) {
	s.endingSoonPayload = &payload
	if s.operationOrderLog != nil {
		*s.operationOrderLog = append(*s.operationOrderLog, "store.auction_ending_soon")
	}
}

func (s *stateStoreSpy) ApplyAuctionEnded(payload events.AuctionEndedPayload) {
	s.endedPayload = &payload
	if s.operationOrderLog != nil {
		*s.operationOrderLog = append(*s.operationOrderLog, "store.auction_ended")
	}
}

func (s *stateStoreSpy) ApplyBidPlaced(payload events.BidPlacedPayload) {
	s.bidPlacedPayload = &payload
	if s.operationOrderLog != nil {
		*s.operationOrderLog = append(*s.operationOrderLog, "store.bid_placed")
	}
}

type broadcastCall struct {
	eventName string
	payload   any
}

type broadcasterSpy struct {
	broadcasts        []broadcastCall
	operationOrderLog *[]string
}

func (b *broadcasterSpy) Broadcast(eventName string, payload any) {
	b.broadcasts = append(b.broadcasts, broadcastCall{eventName: eventName, payload: payload})
	if b.operationOrderLog != nil {
		*b.operationOrderLog = append(*b.operationOrderLog, "broadcast."+eventName)
	}
}

func (b *broadcasterSpy) Subscribe() (<-chan domain.SSEEvent, func()) {
	return nil, func() {}
}

type fakeDelivery struct {
	body        []byte
	routingKey  string
	deliveryTag uint64
	ackCalls    int
	nackCalls   int
	nackRequeue bool
	ackErr      error
	nackErr     error
}

func (d *fakeDelivery) Body() []byte        { return d.body }
func (d *fakeDelivery) RoutingKey() string  { return d.routingKey }
func (d *fakeDelivery) DeliveryTag() uint64 { return d.deliveryTag }

func (d *fakeDelivery) Ack(_ bool) error {
	d.ackCalls++
	return d.ackErr
}

func (d *fakeDelivery) Nack(_ bool, requeue bool) error {
	d.nackCalls++
	d.nackRequeue = requeue
	return d.nackErr
}

func newTestConsumer(store domain.StateStore, broadcaster domain.EventBroadcaster) *BFFEventConsumer {
	return &BFFEventConsumer{
		store:       store,
		broadcaster: broadcaster,
		logger:      zap.NewNop(),
	}
}

func makeDelivery(t *testing.T, eventType string, payload any) *fakeDelivery {
	t.Helper()

	envelope := events.NewEnvelope(eventType, eventVersionFor(eventType), "", payload)
	body, err := json.Marshal(envelope)
	require.NoError(t, err)

	return &fakeDelivery{body: body, routingKey: eventType}
}

func eventVersionFor(eventType string) string {
	if eventType == events.RoutingKeyBidPlaced {
		return events.BidEventVersion
	}
	return events.AuctionEventVersion
}

func makeAuctionCreatedPayload() events.AuctionCreatedPayload {
	now := time.Now().UTC()
	return events.AuctionCreatedPayload{
		AuctionID:   101,
		Title:       "Vintage Camera",
		Description: "Collector-grade rangefinder",
		StartPrice:  150.0,
		StartTime:   now.Format(time.RFC3339),
		EndTime:     now.Add(90 * time.Minute).Format(time.RFC3339),
	}
}

func makeAuctionEndingSoonPayload() events.AuctionEndingSoonPayload {
	return events.AuctionEndingSoonPayload{
		AuctionID: 202,
		EndTime:   time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
}

func makeAuctionEndedPayload() events.AuctionEndedPayload {
	return events.AuctionEndedPayload{
		AuctionID:   303,
		WinnerBotID: 2,
		WinningBid:  245.5,
		TotalBids:   7,
		FinalStatus: "sold",
	}
}

func makeBidPlacedPayload() events.BidPlacedPayload {
	return events.BidPlacedPayload{
		AuctionID: 404,
		BotID:     4,
		BidAmount: 199.99,
		BidID:     9001,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func TestHandleDelivery_MalformedJSON_AcksAndIgnores(t *testing.T) {
	store := &stateStoreSpy{}
	broadcaster := &broadcasterSpy{}
	consumer := newTestConsumer(store, broadcaster)

	delivery := &fakeDelivery{body: []byte("not-json"), routingKey: events.RoutingKeyAuctionCreated}

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
	assert.Nil(t, store.createdPayload)
	assert.Nil(t, store.endingSoonPayload)
	assert.Nil(t, store.endedPayload)
	assert.Nil(t, store.bidPlacedPayload)
	assert.Empty(t, broadcaster.broadcasts)
}

func TestHandleDelivery_MalformedJSON_AckFailureIsReturned(t *testing.T) {
	consumer := newTestConsumer(&stateStoreSpy{}, &broadcasterSpy{})
	delivery := &fakeDelivery{
		body:       []byte("not-json"),
		routingKey: events.RoutingKeyAuctionCreated,
		ackErr:     errors.New("ack failed"),
	}

	err := consumer.handleDelivery(context.Background(), delivery)
	require.Error(t, err)
	assert.ErrorContains(t, err, "ack malformed envelope")
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_UnknownEventType_AcksAndIgnores(t *testing.T) {
	store := &stateStoreSpy{}
	broadcaster := &broadcasterSpy{}
	consumer := newTestConsumer(store, broadcaster)

	delivery := makeDelivery(t, "some.unknown.event", map[string]any{"foo": "bar"})

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
	assert.Nil(t, store.createdPayload)
	assert.Nil(t, store.endingSoonPayload)
	assert.Nil(t, store.endedPayload)
	assert.Nil(t, store.bidPlacedPayload)
	assert.Empty(t, broadcaster.broadcasts)
}

func TestHandleDelivery_SupportedEvents_ApplyToStoreAndBroadcast(t *testing.T) {
	tests := []struct {
		name           string
		eventType      string
		payload        any
		storeAssertion func(t *testing.T, store *stateStoreSpy)
		expectedOrder  []string
	}{
		{
			name:      "auction created",
			eventType: events.RoutingKeyAuctionCreated,
			payload:   makeAuctionCreatedPayload(),
			storeAssertion: func(t *testing.T, store *stateStoreSpy) {
				require.NotNil(t, store.createdPayload)
				assert.Nil(t, store.endingSoonPayload)
				assert.Nil(t, store.endedPayload)
				assert.Nil(t, store.bidPlacedPayload)
			},
			expectedOrder: []string{"store.auction_created", "broadcast." + events.RoutingKeyAuctionCreated},
		},
		{
			name:      "auction ending soon",
			eventType: events.RoutingKeyAuctionEndingSoon,
			payload:   makeAuctionEndingSoonPayload(),
			storeAssertion: func(t *testing.T, store *stateStoreSpy) {
				require.NotNil(t, store.endingSoonPayload)
				assert.Nil(t, store.createdPayload)
				assert.Nil(t, store.endedPayload)
				assert.Nil(t, store.bidPlacedPayload)
			},
			expectedOrder: []string{"store.auction_ending_soon", "broadcast." + events.RoutingKeyAuctionEndingSoon},
		},
		{
			name:      "auction ended",
			eventType: events.RoutingKeyAuctionEnded,
			payload:   makeAuctionEndedPayload(),
			storeAssertion: func(t *testing.T, store *stateStoreSpy) {
				require.NotNil(t, store.endedPayload)
				assert.Nil(t, store.createdPayload)
				assert.Nil(t, store.endingSoonPayload)
				assert.Nil(t, store.bidPlacedPayload)
			},
			expectedOrder: []string{"store.auction_ended", "broadcast." + events.RoutingKeyAuctionEnded},
		},
		{
			name:      "bid placed",
			eventType: events.RoutingKeyBidPlaced,
			payload:   makeBidPlacedPayload(),
			storeAssertion: func(t *testing.T, store *stateStoreSpy) {
				require.NotNil(t, store.bidPlacedPayload)
				assert.Nil(t, store.createdPayload)
				assert.Nil(t, store.endingSoonPayload)
				assert.Nil(t, store.endedPayload)
			},
			expectedOrder: []string{"store.bid_placed", "broadcast." + events.RoutingKeyBidPlaced},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operationOrder := make([]string, 0, 2)
			store := &stateStoreSpy{operationOrderLog: &operationOrder}
			broadcaster := &broadcasterSpy{operationOrderLog: &operationOrder}
			consumer := newTestConsumer(store, broadcaster)
			delivery := makeDelivery(t, tt.eventType, tt.payload)

			err := consumer.handleDelivery(context.Background(), delivery)
			require.NoError(t, err)

			tt.storeAssertion(t, store)
			require.Len(t, broadcaster.broadcasts, 1)
			assert.Equal(t, tt.eventType, broadcaster.broadcasts[0].eventName)
			assert.Equal(t, tt.payload, broadcaster.broadcasts[0].payload)
			assert.Equal(t, 1, delivery.ackCalls)
			assert.Equal(t, 0, delivery.nackCalls)
			assert.Equal(t, tt.expectedOrder, operationOrder)
		})
	}
}

func TestHandleDelivery_NonRetryablePayloads_AreAckedAndDropped(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
	}{
		{name: "auction created", eventType: events.RoutingKeyAuctionCreated},
		{name: "auction ending soon", eventType: events.RoutingKeyAuctionEndingSoon},
		{name: "auction ended", eventType: events.RoutingKeyAuctionEnded},
		{name: "bid placed", eventType: events.RoutingKeyBidPlaced},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &stateStoreSpy{}
			broadcaster := &broadcasterSpy{}
			consumer := newTestConsumer(store, broadcaster)
			delivery := makeDelivery(t, tt.eventType, "bad-payload-shape")

			err := consumer.handleDelivery(context.Background(), delivery)
			require.NoError(t, err)

			assert.Equal(t, 1, delivery.ackCalls)
			assert.Equal(t, 0, delivery.nackCalls)
			assert.Nil(t, store.createdPayload)
			assert.Nil(t, store.endingSoonPayload)
			assert.Nil(t, store.endedPayload)
			assert.Nil(t, store.bidPlacedPayload)
			assert.Empty(t, broadcaster.broadcasts)
		})
	}
}

func TestHandleDelivery_SuccessAckFailureIsReturned(t *testing.T) {
	consumer := newTestConsumer(&stateStoreSpy{}, &broadcasterSpy{})
	delivery := makeDelivery(t, events.RoutingKeyAuctionCreated, makeAuctionCreatedPayload())
	delivery.ackErr = errors.New("ack failed")

	err := consumer.handleDelivery(context.Background(), delivery)
	require.Error(t, err)
	assert.ErrorContains(t, err, "ack message")
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_NonRetryableAckFailureIsReturned(t *testing.T) {
	consumer := newTestConsumer(&stateStoreSpy{}, &broadcasterSpy{})
	delivery := makeDelivery(t, events.RoutingKeyBidPlaced, "bad-payload-shape")
	delivery.ackErr = errors.New("ack failed")

	err := consumer.handleDelivery(context.Background(), delivery)
	require.Error(t, err)
	assert.ErrorContains(t, err, "ack non-retryable error")
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestReUnmarshalPayload(t *testing.T) {
	now := time.Now().UTC()
	raw := map[string]any{
		"auction_id":  float64(5),
		"title":       "Rare Poster",
		"description": "Limited print",
		"start_price": float64(75.0),
		"start_time":  now.Format(time.RFC3339),
		"end_time":    now.Add(time.Hour).Format(time.RFC3339),
	}

	result, err := reUnmarshalPayload[events.AuctionCreatedPayload](raw)

	require.NoError(t, err)
	assert.Equal(t, int64(5), result.AuctionID)
	assert.Equal(t, "Rare Poster", result.Title)
	assert.Equal(t, "Limited print", result.Description)
	assert.Equal(t, 75.0, result.StartPrice)
	assert.Equal(t, raw["start_time"], result.StartTime)
	assert.Equal(t, raw["end_time"], result.EndTime)
}

func TestReUnmarshalPayload_ReMarshalError(t *testing.T) {
	_, err := reUnmarshalPayload[events.AuctionCreatedPayload](make(chan int))

	require.Error(t, err)
	assert.ErrorContains(t, err, "re-marshal payload")
}

func TestReUnmarshalPayload_UnmarshalError(t *testing.T) {
	_, err := reUnmarshalPayload[events.AuctionCreatedPayload]("not-an-object")

	require.Error(t, err)
	assert.ErrorContains(t, err, "unmarshal payload")
}

var _ domain.StateStore = (*stateStoreSpy)(nil)
var _ domain.EventBroadcaster = (*broadcasterSpy)(nil)
