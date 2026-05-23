package mq

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/Osireg17/AI-Bidding-Platform/shared/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeBankingService struct {
	calls       []recordWinCall
	errToReturn error
}

type recordWinCall struct {
	botID      int64
	auctionID  int64
	title      string
	winningBid float64
}

func (f *fakeBankingService) RecordWin(_ context.Context, botID, auctionID int64, title string, winningBid float64) (float64, error) {
	f.calls = append(f.calls, recordWinCall{botID: botID, auctionID: auctionID, title: title, winningBid: winningBid})
	return 0, f.errToReturn
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

func newTestConsumer(svc BankingService) *BankingEventConsumer {
	c := &BankingEventConsumer{svc: svc, logger: zap.NewNop()}
	c.BaseConsumer = messaging.NewBaseConsumerForTest(messaging.ConsumerConfig{}, c, zap.NewNop())
	return c
}

func makeDelivery(t *testing.T, eventType string, payload any) *fakeDelivery {
	t.Helper()
	envelope := events.NewEnvelope(eventType, events.AuctionEventVersion, "", payload)
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	return &fakeDelivery{body: body, routingKey: eventType}
}

// ---------------------------------------------------------------------------
// handleAuctionEnded — unmarshal failure
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionEnded_MalformedBody_IsAckedAndDropped(t *testing.T) {
	svc := &fakeBankingService{}
	consumer := newTestConsumer(svc)
	d := &fakeDelivery{body: []byte("not-json"), routingKey: events.RoutingKeyAuctionEnded}

	err := consumer.BaseConsumer.HandleDelivery(context.Background(), d)
	require.NoError(t, err)
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	assert.Empty(t, svc.calls)
}

// ---------------------------------------------------------------------------
// handleAuctionEnded — skip conditions
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionEnded_Unsold_SkipsRecordWin(t *testing.T) {
	svc := &fakeBankingService{}
	consumer := newTestConsumer(svc)
	payload := events.AuctionEndedPayload{AuctionID: 1, FinalStatus: "unsold", WinnerBotID: 0}
	d := makeDelivery(t, events.RoutingKeyAuctionEnded, payload)

	err := consumer.BaseConsumer.HandleDelivery(context.Background(), d)
	require.NoError(t, err)
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	assert.Empty(t, svc.calls)
}

func TestHandleDelivery_AuctionEnded_SoldButNoWinner_SkipsRecordWin(t *testing.T) {
	svc := &fakeBankingService{}
	consumer := newTestConsumer(svc)
	payload := events.AuctionEndedPayload{AuctionID: 1, FinalStatus: "sold", WinnerBotID: 0}
	d := makeDelivery(t, events.RoutingKeyAuctionEnded, payload)

	err := consumer.BaseConsumer.HandleDelivery(context.Background(), d)
	require.NoError(t, err)
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	assert.Empty(t, svc.calls)
}

// ---------------------------------------------------------------------------
// handleAuctionEnded — happy path
// ---------------------------------------------------------------------------

func TestHandleDelivery_AuctionEnded_Sold_CallsRecordWinWithCorrectArgs(t *testing.T) {
	svc := &fakeBankingService{}
	consumer := newTestConsumer(svc)
	payload := events.AuctionEndedPayload{
		AuctionID:   10,
		WinnerBotID: 2,
		WinningBid:  500.0,
		Title:       "Vintage Watch",
		FinalStatus: "sold",
	}
	d := makeDelivery(t, events.RoutingKeyAuctionEnded, payload)

	err := consumer.BaseConsumer.HandleDelivery(context.Background(), d)
	require.NoError(t, err)
	assert.Equal(t, 1, d.ackCalls)
	assert.Equal(t, 0, d.nackCalls)
	require.Len(t, svc.calls, 1)
	call := svc.calls[0]
	assert.Equal(t, int64(2), call.botID)
	assert.Equal(t, int64(10), call.auctionID)
	assert.Equal(t, "Vintage Watch", call.title)
	assert.Equal(t, 500.0, call.winningBid)
}

var _ BankingService = (*fakeBankingService)(nil)
var _ messaging.Delivery = (*fakeDelivery)(nil)
