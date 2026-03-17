package http

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- fakes ---

type fakeStore struct {
	state domain.AuctionState
}

func (f *fakeStore) GetState() domain.AuctionState                            { return f.state }
func (f *fakeStore) ApplyAuctionCreated(_ events.AuctionCreatedPayload)       {}
func (f *fakeStore) ApplyAuctionEndingSoon(_ events.AuctionEndingSoonPayload) {}
func (f *fakeStore) ApplyAuctionEnded(_ events.AuctionEndedPayload)           {}
func (f *fakeStore) ApplyBidPlaced(_ events.BidPlacedPayload)                 {}

type fakeBroadcaster struct {
	ch          chan domain.SSEEvent
	unsubscribe func()
}

func newFakeBroadcaster() *fakeBroadcaster {
	ch := make(chan domain.SSEEvent, 8)
	return &fakeBroadcaster{
		ch:          ch,
		unsubscribe: func() {},
	}
}

func (f *fakeBroadcaster) Broadcast(_ string, _ any) {}
func (f *fakeBroadcaster) Subscribe() (<-chan domain.SSEEvent, func()) {
	return f.ch, f.unsubscribe
}

// --- helpers ---

func newTestHandler(store domain.StateStore, broadcaster domain.EventBroadcaster) *BFFHandler {
	return NewBFFHandler(store, broadcaster, zap.NewNop())
}

func newGinContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func decodeStateResponse(t *testing.T, w *httptest.ResponseRecorder) stateResponse {
	t.Helper()
	var resp stateResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

// --- HandleGetState ---

func TestHandleGetState_NoAuction(t *testing.T) {
	store := &fakeStore{state: domain.AuctionState{}}
	handler := newTestHandler(store, newFakeBroadcaster())

	c, w := newGinContext(http.MethodGet, "/api/state")
	handler.HandleGetState(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeStateResponse(t, w)
	assert.Equal(t, stateResponse{
		Auction: nil,
		Bids:    []bidJSON{},
		Winner:  nil,
	}, resp)
}

func TestHandleGetState_ActiveAuction(t *testing.T) {
	endTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	bidTime := endTime.Add(-10 * time.Minute)

	store := &fakeStore{state: domain.AuctionState{
		HasAuction: true,
		Auction: domain.AuctionView{
			ID:           42,
			Title:        "Rare Painting",
			Description:  "Oil on canvas",
			StartPrice:   100.0,
			CurrentPrice: 150.0,
			Status:       "active",
			EndTime:      endTime,
		},
		Bids: []domain.BidView{
			{BotName: "Aggressive Alice", BotID: 1, Amount: 150.0, Timestamp: bidTime},
		},
	}}
	handler := newTestHandler(store, newFakeBroadcaster())

	c, w := newGinContext(http.MethodGet, "/api/state")
	handler.HandleGetState(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeStateResponse(t, w)
	assert.Equal(t, stateResponse{
		Auction: &auctionJSON{
			ID:           42,
			Title:        "Rare Painting",
			Description:  "Oil on canvas",
			StartPrice:   100.0,
			CurrentPrice: 150.0,
			Status:       "active",
			EndTime:      endTime.Format(time.RFC3339),
		},
		Bids: []bidJSON{
			{BotName: "Aggressive Alice", BotID: 1, Amount: 150.0, Timestamp: bidTime.Format(time.RFC3339)},
		},
		Winner: nil,
	}, resp)
}

func TestHandleGetState_WithWinner(t *testing.T) {
	store := &fakeStore{state: domain.AuctionState{
		HasAuction: true,
		Auction:    domain.AuctionView{ID: 1, Status: "closed", EndTime: time.Time{}},
		HasWinner:  true,
		Winner: domain.WinnerView{
			BotName:     "Sniper Steve",
			BotID:       2,
			Amount:      500.0,
			FinalStatus: "sold",
		},
	}}
	handler := newTestHandler(store, newFakeBroadcaster())

	c, w := newGinContext(http.MethodGet, "/api/state")
	handler.HandleGetState(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeStateResponse(t, w)
	assert.Equal(t, stateResponse{
		Auction: &auctionJSON{
			ID:      1,
			Status:  "closed",
			EndTime: time.Time{}.Format(time.RFC3339),
		},
		Bids: []bidJSON{},
		Winner: &winnerJSON{
			BotName:     "Sniper Steve",
			BotID:       2,
			Amount:      500.0,
			FinalStatus: "sold",
		},
	}, resp)
}

// --- HandleStream ---

func TestHandleStream_SendsSnapshotOnConnect(t *testing.T) {
	store := &fakeStore{state: domain.AuctionState{
		HasAuction: true,
		Auction:    domain.AuctionView{ID: 7, Title: "Live Auction", Status: "active"},
	}}
	fb := newFakeBroadcaster()
	close(fb.ch)

	handler := newTestHandler(store, fb)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)

	handler.HandleStream(c)

	body := w.Body.String()
	assert.Contains(t, body, "event: auction.snapshot")
	assert.Contains(t, body, `"id":7`)
	assert.Contains(t, body, `"title":"Live Auction"`)
}

func TestHandleStream_ForwardsEventFromBroadcaster(t *testing.T) {
	store := &fakeStore{}
	fb := newFakeBroadcaster()
	handler := newTestHandler(store, fb)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)

	go func() {
		fb.ch <- domain.SSEEvent{Name: "bid.placed", Payload: []byte(`{"amount":200}`)}
		cancel()
	}()

	handler.HandleStream(c)

	body := w.Body.String()
	assert.Contains(t, body, "event: bid.placed")
	assert.Contains(t, body, `"amount":200`)
}

func TestHandleStream_SSEHeaders(t *testing.T) {
	store := &fakeStore{}
	fb := newFakeBroadcaster()
	close(fb.ch)

	handler := newTestHandler(store, fb)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)

	handler.HandleStream(c)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

func TestHandleStream_EventFormat(t *testing.T) {
	store := &fakeStore{}
	fb := newFakeBroadcaster()
	handler := newTestHandler(store, fb)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)

	go func() {
		fb.ch <- domain.SSEEvent{Name: "auction.ended", Payload: []byte(`{"status":"sold"}`)}
		cancel()
	}()

	handler.HandleStream(c)

	// Parse lines and verify "event: <name>" is immediately followed by "data: ..."
	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	foundValidBlock := false
	for i, line := range lines {
		if line == "event: auction.ended" && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "data: ") {
			foundValidBlock = true
			break
		}
	}
	assert.True(t, foundValidBlock, "expected 'event: auction.ended' followed by 'data: ...' line, got:\n%s", w.Body.String())
}
