package mq

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testPublisher *RabbitMQPublisher
	testConn      *amqp.Connection
	testChannel   *amqp.Channel
)

func TestMain(m *testing.M) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	logger := zap.NewNop()
	publisher, err := NewRabbitMQPublisher(rabbitURL, logger)
	if err != nil {
		panic("failed to create publisher: " + err.Error())
	}
	testPublisher = publisher

	testConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		panic("failed to connect to RabbitMQ: " + err.Error())
	}

	testChannel, err = testConn.Channel()
	if err != nil {
		panic("failed to open channel: " + err.Error())
	}

	code := m.Run()

	testPublisher.Close()
	testChannel.Close()
	testConn.Close()

	os.Exit(code)
}

// consumeOne declares a temp queue bound to the exchange with the given routing key,
// starts consuming, and returns the deliveries channel plus the queue name for cleanup.
func consumeOne(t *testing.T, routingKey string) (<-chan amqp.Delivery, string) {
	t.Helper()

	q, err := testChannel.QueueDeclare(
		"",    // empty name = server-generated
		false, // not durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,
	)
	require.NoError(t, err, "failed to declare temp queue")

	err = testChannel.QueueBind(
		q.Name,
		routingKey,
		events.ExchangeName,
		false, // no-wait
		nil,
	)
	require.NoError(t, err, "failed to bind temp queue")

	deliveries, err := testChannel.Consume(
		q.Name,
		"",    // consumer tag (server-generated)
		true,  // auto-ack
		true,  // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	require.NoError(t, err, "failed to start consuming")

	return deliveries, q.Name
}

// awaitMessage waits up to 2 seconds for a message on the deliveries channel.
func awaitMessage(t *testing.T, deliveries <-chan amqp.Delivery) amqp.Delivery {
	t.Helper()
	select {
	case msg := <-deliveries:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
		return amqp.Delivery{} // unreachable, keeps compiler happy
	}
}

func unmarshalEnvelope(t *testing.T, body []byte) events.Envelope {
	t.Helper()
	var env events.Envelope
	err := json.Unmarshal(body, &env)
	require.NoError(t, err, "failed to unmarshal envelope")
	return env
}

func extractPayload(t *testing.T, rawPayload interface{}, target interface{}) {
	t.Helper()
	payloadBytes, err := json.Marshal(rawPayload)
	require.NoError(t, err, "failed to re-marshal payload")
	err = json.Unmarshal(payloadBytes, target)
	require.NoError(t, err, "failed to unmarshal payload into target")
}

func newTestAuction() *domain.Auction {
	now := time.Now().UTC()
	return &domain.Auction{
		ID:          42,
		Title:       "Test Auction",
		Description: "A test",
		StartPrice:  100.50,
		Status:      domain.StatusActive,
		StartTime:   now,
		EndTime:     now.Add(1 * time.Hour),
	}
}

func TestPublishAuctionCreated(t *testing.T) {
	deliveries, qName := consumeOne(t, events.RoutingKeyAuctionCreated)
	defer testChannel.QueueDelete(qName, false, false, false)

	auction := newTestAuction()
	err := testPublisher.PublishAuctionCreated(context.Background(), auction)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)
	env := unmarshalEnvelope(t, msg.Body)

	// Envelope-level assertions.
	assert.Equal(t, events.RoutingKeyAuctionCreated, env.EventType)
	assert.Equal(t, events.AuctionEventVersion, env.EventVersion)
	assert.NotEmpty(t, env.EventID)
	assert.WithinDuration(t, time.Now().UTC(), env.OccurredAt, 5*time.Second)

	// Payload assertions.
	var payload events.AuctionCreatedPayload
	extractPayload(t, env.Payload, &payload)

	assert.Equal(t, int64(42), payload.AuctionID)
	assert.Equal(t, "Test Auction", payload.Title)
	assert.Equal(t, "A test", payload.Description)
	assert.Equal(t, 100.50, payload.StartPrice)
	assert.NotEmpty(t, payload.StartTime)
	assert.NotEmpty(t, payload.EndTime)
}

func TestPublishAuctionEndingSoon(t *testing.T) {
	deliveries, qName := consumeOne(t, events.RoutingKeyAuctionEndingSoon)
	defer testChannel.QueueDelete(qName, false, false, false)

	auction := newTestAuction()
	err := testPublisher.PublishAuctionEndingSoon(context.Background(), auction)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)
	env := unmarshalEnvelope(t, msg.Body)

	assert.Equal(t, events.RoutingKeyAuctionEndingSoon, env.EventType)
	assert.Equal(t, events.AuctionEventVersion, env.EventVersion)
	assert.NotEmpty(t, env.EventID)

	var payload events.AuctionEndingSoonPayload
	extractPayload(t, env.Payload, &payload)

	assert.Equal(t, int64(42), payload.AuctionID)
	assert.NotEmpty(t, payload.EndTime)
}

func TestPublishAuctionEnded_Sold(t *testing.T) {
	deliveries, qName := consumeOne(t, events.RoutingKeyAuctionEnded)
	defer testChannel.QueueDelete(qName, false, false, false)

	auction := newTestAuction()
	auction.WinnerBotID = 7
	auction.CurrentPrice = 250.75

	err := testPublisher.PublishAuctionEnded(context.Background(), auction)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)
	env := unmarshalEnvelope(t, msg.Body)

	assert.Equal(t, events.RoutingKeyAuctionEnded, env.EventType)

	var payload events.AuctionEndedPayload
	extractPayload(t, env.Payload, &payload)

	assert.Equal(t, int64(42), payload.AuctionID)
	assert.Equal(t, int64(7), payload.WinnerBotID)
	assert.Equal(t, 250.75, payload.WinningBid)
	assert.Equal(t, "sold", payload.FinalStatus)
}

func TestPublishAuctionEnded_Unsold(t *testing.T) {
	deliveries, qName := consumeOne(t, events.RoutingKeyAuctionEnded)
	defer testChannel.QueueDelete(qName, false, false, false)

	auction := newTestAuction()
	// WinnerBotID stays 0 → "unsold"

	err := testPublisher.PublishAuctionEnded(context.Background(), auction)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)
	env := unmarshalEnvelope(t, msg.Body)

	var payload events.AuctionEndedPayload
	extractPayload(t, env.Payload, &payload)

	assert.Equal(t, int64(42), payload.AuctionID)
	assert.Equal(t, int64(0), payload.WinnerBotID)
	assert.Equal(t, 0.0, payload.WinningBid)
	assert.Equal(t, "unsold", payload.FinalStatus)
}

func TestNewRabbitMQPublisher_InvalidURL(t *testing.T) {
	logger := zap.NewNop()
	_, err := NewRabbitMQPublisher("amqp://invalid:invalid@localhost:9999/", logger)
	require.Error(t, err)
}

func TestClose(t *testing.T) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	logger := zap.NewNop()
	pub, err := NewRabbitMQPublisher(rabbitURL, logger)
	require.NoError(t, err)

	err = pub.Close()
	assert.NoError(t, err)
}
