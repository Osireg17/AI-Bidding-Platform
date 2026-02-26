package mq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock ---

type mockSnapshotRepo struct {
	upsertCalled       bool
	upsertSnapshot     *domain.AuctionSnapshot
	updateStatusCalled bool
	updateStatusID     int64
	updateStatusValue  domain.AuctionStatus
	errToReturn        error
}

func (m *mockSnapshotRepo) Upsert(_ context.Context, snapshot *domain.AuctionSnapshot) error {
	m.upsertCalled = true
	m.upsertSnapshot = snapshot
	return m.errToReturn
}

func (m *mockSnapshotRepo) GetByID(_ context.Context, _ int64) (*domain.AuctionSnapshot, error) {
	return nil, m.errToReturn
}

func (m *mockSnapshotRepo) UpdateStatus(_ context.Context, auctionID int64, status domain.AuctionStatus) error {
	m.updateStatusCalled = true
	m.updateStatusID = auctionID
	m.updateStatusValue = status
	return m.errToReturn
}

// --- Helpers ---

func newTestConsumer(repo domain.AuctionSnapshotRepository) *AuctionEventConsumer {
	return &AuctionEventConsumer{conn: nil, channel: nil, repo: repo, logger: zap.NewNop()}
}

type fakeDelivery struct {
	body        []byte
	routingKey  string
	deliveryTag uint64

	ackCalls  int
	nackCalls int

	nackRequeue bool
}

func (d *fakeDelivery) Body() []byte {
	return d.body
}

func (d *fakeDelivery) RoutingKey() string {
	return d.routingKey
}

func (d *fakeDelivery) DeliveryTag() uint64 {
	return d.deliveryTag
}

func (d *fakeDelivery) Ack(_ bool) error {
	d.ackCalls++
	return nil
}

func (d *fakeDelivery) Nack(_ bool, requeue bool) error {
	d.nackCalls++
	d.nackRequeue = requeue
	return nil
}

func makeDelivery(t *testing.T, eventType string, payload any) *fakeDelivery {
	t.Helper()
	envelope := events.NewEnvelope(eventType, events.AuctionEventVersion, "", payload)
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	return &fakeDelivery{
		body:       body,
		routingKey: eventType,
	}
}

func makeAuctionCreatedPayload() events.AuctionCreatedPayload {
	now := time.Now().UTC()
	return events.AuctionCreatedPayload{
		AuctionID:   1,
		Title:       "Test Auction",
		Description: "A test description",
		StartPrice:  50.0,
		StartTime:   now.Format(time.RFC3339),
		EndTime:     now.Add(time.Hour).Format(time.RFC3339),
	}
}

// --- Tests ---

func TestHandleDelivery_AuctionCreated(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	payload := makeAuctionCreatedPayload()
	delivery := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	require.True(t, repo.upsertCalled)
	require.NotNil(t, repo.upsertSnapshot)
	assert.Equal(t, payload.AuctionID, repo.upsertSnapshot.AuctionID)
	assert.Equal(t, domain.AuctionStatusActive, repo.upsertSnapshot.Status)
	assert.Equal(t, payload.Title, repo.upsertSnapshot.Title)
	assert.Equal(t, payload.StartPrice, repo.upsertSnapshot.StartPrice)
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_AuctionEndingSoon(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	payload := events.AuctionEndingSoonPayload{
		AuctionID: 2,
		EndTime:   time.Now().UTC().Format(time.RFC3339),
	}
	delivery := makeDelivery(t, events.RoutingKeyAuctionEndingSoon, payload)

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	require.True(t, repo.updateStatusCalled)
	assert.Equal(t, int64(2), repo.updateStatusID)
	assert.Equal(t, domain.AuctionStatusEndingSoon, repo.updateStatusValue)
	assert.False(t, repo.upsertCalled)
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_AuctionEnded(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	payload := events.AuctionEndedPayload{
		AuctionID:   3,
		FinalStatus: "sold",
		TotalBids:   5,
	}
	delivery := makeDelivery(t, events.RoutingKeyAuctionEnded, payload)

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	require.True(t, repo.updateStatusCalled)
	assert.Equal(t, int64(3), repo.updateStatusID)
	assert.Equal(t, domain.AuctionStatusClosed, repo.updateStatusValue)
	assert.False(t, repo.upsertCalled)
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_UnknownEventType(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	delivery := makeDelivery(t, "some.unknown.event", map[string]any{"foo": "bar"})

	err := consumer.handleDelivery(context.Background(), delivery)
	require.NoError(t, err)

	assert.False(t, repo.upsertCalled)
	assert.False(t, repo.updateStatusCalled)
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_MalformedJSON(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	delivery := &fakeDelivery{
		body:       []byte("not-json"),
		routingKey: "auction.created",
	}

	assert.NotPanics(t, func() {
		err := consumer.handleDelivery(context.Background(), delivery)
		require.NoError(t, err)
	})
	assert.False(t, repo.upsertCalled)
	assert.False(t, repo.updateStatusCalled)
	assert.Equal(t, 1, delivery.ackCalls)
	assert.Equal(t, 0, delivery.nackCalls)
}

func TestHandleDelivery_RepoError(t *testing.T) {
	repo := &mockSnapshotRepo{errToReturn: errors.New("db down")}
	consumer := newTestConsumer(repo)
	payload := makeAuctionCreatedPayload()
	delivery := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	assert.NotPanics(t, func() {
		err := consumer.handleDelivery(context.Background(), delivery)
		require.NoError(t, err)
	})
	assert.True(t, repo.upsertCalled)
	assert.Equal(t, 0, delivery.ackCalls)
	assert.Equal(t, 1, delivery.nackCalls)
	assert.True(t, delivery.nackRequeue)
}

func TestReUnmarshalPayload(t *testing.T) {
	now := time.Now().UTC()
	raw := map[string]any{
		"auction_id":  float64(5),
		"title":       "hello",
		"description": "world",
		"start_price": float64(100.0),
		"start_time":  now.Format(time.RFC3339),
		"end_time":    now.Add(time.Hour).Format(time.RFC3339),
	}

	result, err := reUnmarshalPayload[events.AuctionCreatedPayload](raw)

	require.NoError(t, err)
	assert.Equal(t, int64(5), result.AuctionID)
	assert.Equal(t, "hello", result.Title)
	assert.Equal(t, "world", result.Description)
	assert.Equal(t, 100.0, result.StartPrice)
}

// Compile-time guard: ensure mockSnapshotRepo satisfies the interface.
var _ domain.AuctionSnapshotRepository = (*mockSnapshotRepo)(nil)
