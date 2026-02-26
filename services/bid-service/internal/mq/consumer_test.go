package mq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
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
	updateStatusValue  string
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

func (m *mockSnapshotRepo) UpdateStatus(_ context.Context, auctionID int64, status string) error {
	m.updateStatusCalled = true
	m.updateStatusID = auctionID
	m.updateStatusValue = status
	return m.errToReturn
}

// --- Helpers ---

func newTestConsumer(repo domain.AuctionSnapshotRepository) *AuctionEventConsumer {
	return &AuctionEventConsumer{conn: nil, channel: nil, repo: repo, logger: zap.NewNop()}
}

func makeDelivery(t *testing.T, eventType string, payload any) amqp.Delivery {
	t.Helper()
	envelope := events.NewEnvelope(eventType, events.AuctionEventVersion, "", payload)
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	return amqp.Delivery{Body: body}
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

	consumer.handleDelivery(context.Background(), delivery)

	require.True(t, repo.upsertCalled)
	require.NotNil(t, repo.upsertSnapshot)
	assert.Equal(t, payload.AuctionID, repo.upsertSnapshot.AuctionID)
	assert.Equal(t, "active", repo.upsertSnapshot.Status)
	assert.Equal(t, payload.Title, repo.upsertSnapshot.Title)
	assert.Equal(t, payload.StartPrice, repo.upsertSnapshot.StartPrice)
}

func TestHandleDelivery_AuctionEndingSoon(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	payload := events.AuctionEndingSoonPayload{
		AuctionID: 2,
		EndTime:   time.Now().UTC().Format(time.RFC3339),
	}
	delivery := makeDelivery(t, events.RoutingKeyAuctionEndingSoon, payload)

	consumer.handleDelivery(context.Background(), delivery)

	require.True(t, repo.updateStatusCalled)
	assert.Equal(t, int64(2), repo.updateStatusID)
	assert.Equal(t, "ending_soon", repo.updateStatusValue)
	assert.False(t, repo.upsertCalled)
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

	consumer.handleDelivery(context.Background(), delivery)

	require.True(t, repo.updateStatusCalled)
	assert.Equal(t, int64(3), repo.updateStatusID)
	assert.Equal(t, "closed", repo.updateStatusValue)
	assert.False(t, repo.upsertCalled)
}

func TestHandleDelivery_UnknownEventType(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	delivery := makeDelivery(t, "some.unknown.event", map[string]any{"foo": "bar"})

	consumer.handleDelivery(context.Background(), delivery)

	assert.False(t, repo.upsertCalled)
	assert.False(t, repo.updateStatusCalled)
}

func TestHandleDelivery_MalformedJSON(t *testing.T) {
	repo := &mockSnapshotRepo{}
	consumer := newTestConsumer(repo)
	delivery := amqp.Delivery{Body: []byte("not-json")}

	assert.NotPanics(t, func() {
		consumer.handleDelivery(context.Background(), delivery)
	})
	assert.False(t, repo.upsertCalled)
	assert.False(t, repo.updateStatusCalled)
}

func TestHandleDelivery_RepoError(t *testing.T) {
	repo := &mockSnapshotRepo{errToReturn: errors.New("db down")}
	consumer := newTestConsumer(repo)
	payload := makeAuctionCreatedPayload()
	delivery := makeDelivery(t, events.RoutingKeyAuctionCreated, payload)

	assert.NotPanics(t, func() {
		consumer.handleDelivery(context.Background(), delivery)
	})
	assert.True(t, repo.upsertCalled)
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
