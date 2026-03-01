package bidclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*BidServiceClient, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewBidServiceClient(server.URL, zaptest.NewLogger(t))
	return client, server
}

func TestPlaceBid_Success(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/bids", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req placeBidRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, int64(1), req.AuctionID)
		assert.Equal(t, int64(2), req.BotID)
		assert.Equal(t, 50.0, req.Amount)

		w.WriteHeader(http.StatusCreated)
	})

	err := client.PlaceBid(context.Background(), 1, 2, 50.0)
	assert.NoError(t, err)
}

func TestPlaceBid_AuctionNotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "auction not found"})
	})

	err := client.PlaceBid(context.Background(), 99, 1, 50.0)
	assert.ErrorIs(t, err, ErrAuctionNotFound)
}

func TestPlaceBid_BidRejected(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bid amount is too low"})
	})

	err := client.PlaceBid(context.Background(), 1, 1, 5.0)
	assert.ErrorIs(t, err, ErrBidRejected)
}

func TestPlaceBid_ServiceUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "lock acquisition failed"})
	})

	err := client.PlaceBid(context.Background(), 1, 1, 50.0)
	assert.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestPlaceBid_UnexpectedStatus(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
	})

	err := client.PlaceBid(context.Background(), 1, 1, 50.0)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrAuctionNotFound)
	assert.NotErrorIs(t, err, ErrBidRejected)
	assert.NotErrorIs(t, err, ErrServiceUnavailable)
}
