package mq

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testPublisher *BidPublisher
	testConn      *amqp.Connection
	testChannel   *amqp.Channel
)

func TestMain(m *testing.M) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	logger := zap.NewNop()
	publisher, err := NewBidPublisher(rabbitURL, logger)
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

	if err := testPublisher.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close testPublisher: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	if err := testChannel.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close testChannel: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	if err := testConn.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close testConn: %v\n", err)
		if code == 0 {
			code = 1
		}
	}

	os.Exit(code)
}

func consumeOne(t *testing.T, routingKey string) (<-chan amqp.Delivery, string) {
	t.Helper()

	q, err := testChannel.QueueDeclare(
		"",    // empty name = server-generated
		false, // not durable
		true,  // auto-delete
		false, // not exclusive
		false, // no-wait
		nil,   // arguments
	)
	require.NoError(t, err)

	err = testChannel.QueueBind(
		q.Name,              // queue name
		routingKey,          // routing key
		events.ExchangeName, // exchange
		false,
		nil,
	)
	require.NoError(t, err)

	deliveries, err := testChannel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	require.NoError(t, err)

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

func extractPayload(t *testing.T, rawPayload any, target any) {
	t.Helper()
	payloadBytes, err := json.Marshal(rawPayload)
	require.NoError(t, err, "failed to re-marshal payload")
	err = json.Unmarshal(payloadBytes, target)
	require.NoError(t, err, "failed to unmarshal payload into target")
}

func newTestBid(t *testing.T) *domain.Bid {
	t.Helper()
	now := time.Now().UTC()
	return &domain.Bid{
		ID:        123,
		AuctionID: 42,
		BotID:     7,
		Amount:    99.99,
		Status:    domain.StatusAccepted,
		Reason:    "",
		CreatedAt: now,
	}
}

func TestPublishBidPlaced(t *testing.T) {
	bid := newTestBid(t)

	deliveries, _ := consumeOne(t, events.RoutingKeyBidPlaced)

	err := testPublisher.PublishBidPlaced(t.Context(), bid)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)

	env := unmarshalEnvelope(t, msg.Body)
	require.Equal(t, events.RoutingKeyBidPlaced, env.EventType)
	require.Equal(t, events.BidEventVersion, env.EventVersion)
	require.NotEmpty(t, env.EventID)
	require.False(t, env.OccurredAt.IsZero())

	var payload events.BidPlacedPayload
	extractPayload(t, env.Payload, &payload)
	require.Equal(t, bid.ID, payload.BidID)
	require.Equal(t, bid.AuctionID, payload.AuctionID)
	require.Equal(t, bid.BotID, payload.BotID)
	require.Equal(t, bid.Amount, payload.BidAmount)
	require.Equal(t, bid.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), payload.Timestamp)
}

func TestPublishBidRejected(t *testing.T) {
	bid := newTestBid(t)
	reason := "bid too low"

	deliveries, _ := consumeOne(t, events.RoutingKeyBidRejected)

	err := testPublisher.PublishBidRejected(t.Context(), bid, reason)
	require.NoError(t, err)

	msg := awaitMessage(t, deliveries)

	env := unmarshalEnvelope(t, msg.Body)
	require.Equal(t, events.RoutingKeyBidRejected, env.EventType)
	require.Equal(t, events.BidEventVersion, env.EventVersion)
	require.NotEmpty(t, env.EventID)
	require.False(t, env.OccurredAt.IsZero())

	var payload events.BidRejectedPayload
	extractPayload(t, env.Payload, &payload)
	require.Equal(t, bid.ID, payload.BidID)
	require.Equal(t, bid.AuctionID, payload.AuctionID)
	require.Equal(t, bid.BotID, payload.BotID)
	require.Equal(t, bid.Amount, payload.BidAmount)
	require.Equal(t, reason, payload.Reason)
	require.Equal(t, bid.CreatedAt.Format(time.RFC3339), payload.Timestamp)
}
