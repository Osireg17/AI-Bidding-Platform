package bankingclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*BankingServiceClient, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewBankingServiceClient(server.URL, zaptest.NewLogger(t))
	return client, server
}

func TestGetWallet_Success(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/wallets/1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"bot_id":1,"balance":100.0,"items":[{"id":1,"title":"Item 1","purchase_price":50.0,"auction_id":1}]}`))
	})
	wallet, err := client.GetWallet(context.Background(), 1)
	assert.NoError(t, err)

	expected := &WalletResponse{
		BotID:   1,
		Balance: 100.0,
		Items: []ItemSummary{
			{
				ID:            1,
				Title:         "Item 1",
				PurchasePrice: 50.0,
				AuctionID:     1,
			},
		},
	}

	assert.Equal(t, expected, wallet)
}

func TestGetWallet_NotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/wallets/99", r.URL.Path)

		w.WriteHeader(http.StatusNotFound)
	})
	_, err := client.GetWallet(context.Background(), 99)
	assert.ErrorIs(t, err, ErrWalletNotFound)
}

func TestGetWallet_ServiceUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/wallets/1", r.URL.Path)

		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := client.GetWallet(context.Background(), 1)
	assert.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestGetWallet_InvalidJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/wallets/1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	})
	_, err := client.GetWallet(context.Background(), 1)
	assert.Error(t, err)
}

func TestBuyoutItem_Success(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/banking/buyout/1", r.URL.Path)

		// respond
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"new_balance":0.0}`))
	})

	newBalance, err := client.Buyout(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, newBalance)
}

func TestBuyoutItem_NotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/banking/buyout/99", r.URL.Path)

		w.WriteHeader(http.StatusNotFound)
	})
	_, err := client.Buyout(context.Background(), 99)
	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestBuyoutItem_ServiceUnavailable(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/banking/buyout/1", r.URL.Path)

		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := client.Buyout(context.Background(), 1)
	assert.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestBuyoutItem_InvalidJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/banking/buyout/1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	})
	_, err := client.Buyout(context.Background(), 1)
	assert.Error(t, err)
}
