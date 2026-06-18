package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/service"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newHandler(svc *service.BankingService, logger *zap.Logger) *BankingHandler {
	return NewBankingHandler(svc, logger)
}

func newGinContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestBankingHandler_GetWallet(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(1), testutil.WithWalletBalance(1500.50))
		item := testutil.CreateTestItem(testutil.WithItemID(10), testutil.WithItemBotID(1), testutil.WithItemPurchasePrice(500))

		walletRepo := &testutil.MockWalletRepository{
			GetByBotIDResult: wallet,
		}
		itemRepo := &testutil.MockItemRepository{
			ListByBotIDResult: []*domain.Item{item},
		}

		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodGet, "/api/v1/wallets/1", nil)
		c.Params = gin.Params{gin.Param{Key: "bot_id", Value: "1"}}

		handler.GetWallet(c)

		require.Equal(t, http.StatusOK, recorder.Code)

		var resp domain.WalletResponse
		decodeJSON(t, recorder, &resp)

		expectedResp := domain.WalletResponse{
			BotID:   1,
			Balance: 1500.50,
			Items: []*domain.ItemSummary{
				{
					ItemID:        item.ID,
					Title:         item.Title,
					PurchasePrice: item.PurchasePrice,
					AcquiredAt:    item.AcquiredAt.Format("2006-01-02 15:04:05"),
				},
			},
		}
		assert.Equal(t, expectedResp, resp)
	})

	t.Run("Invalid Bot ID", func(t *testing.T) {
		logger := zap.NewNop()
		handler := newHandler(nil, logger)

		c, recorder := newGinContext(http.MethodGet, "/api/v1/wallets/abc", nil)
		c.Params = gin.Params{gin.Param{Key: "bot_id", Value: "abc"}}

		handler.GetWallet(c)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "invalid bot_id"}, resp)
	})

	t.Run("Wallet Not Found", func(t *testing.T) {
		walletRepo := &testutil.MockWalletRepository{
			GetByBotIDResult: nil, // Simulates not found when err is nil but wallet is nil
		}
		itemRepo := &testutil.MockItemRepository{}
		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodGet, "/api/v1/wallets/999", nil)
		c.Params = gin.Params{gin.Param{Key: "bot_id", Value: "999"}}

		handler.GetWallet(c)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "wallet not found"}, resp)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		walletRepo := &testutil.MockWalletRepository{
			GetByBotIDErr: errors.New("db connection failed"),
		}
		itemRepo := &testutil.MockItemRepository{}
		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodGet, "/api/v1/wallets/1", nil)
		c.Params = gin.Params{gin.Param{Key: "bot_id", Value: "1"}}

		handler.GetWallet(c)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "internal server error"}, resp)
	})
}

func TestBankingHandler_Buyout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(1), testutil.WithWalletBalance(1000))
		item := testutil.CreateTestItem(testutil.WithItemID(10), testutil.WithItemBotID(1), testutil.WithItemPurchasePrice(100))

		walletRepo := &testutil.MockWalletRepository{
			GetByBotIDResult: wallet,
		}
		itemRepo := &testutil.MockItemRepository{
			GetByIDResult: item,
		}

		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodPost, "/api/v1/items/10/buyout", nil)
		c.Params = gin.Params{gin.Param{Key: "item_id", Value: "10"}}

		handler.Buyout(c)

		require.Equal(t, http.StatusOK, recorder.Code)

		var resp map[string]float64
		decodeJSON(t, recorder, &resp)

		expectedResp := map[string]float64{
			"new_balance": 1000.0 + (100.0 * 0.70),
		}
		assert.Equal(t, expectedResp, resp)
	})

	t.Run("Invalid Item ID", func(t *testing.T) {
		logger := zap.NewNop()
		handler := newHandler(nil, logger)

		c, recorder := newGinContext(http.MethodPost, "/api/v1/items/abc/buyout", nil)
		c.Params = gin.Params{gin.Param{Key: "item_id", Value: "abc"}}

		handler.Buyout(c)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "invalid item_id"}, resp)
	})

	t.Run("Item Not Found", func(t *testing.T) {
		walletRepo := &testutil.MockWalletRepository{}
		itemRepo := &testutil.MockItemRepository{
			GetByIDResult: nil, // Simulates item not found
		}
		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodPost, "/api/v1/items/999/buyout", nil)
		c.Params = gin.Params{gin.Param{Key: "item_id", Value: "999"}}

		handler.Buyout(c)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "item not found"}, resp)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		walletRepo := &testutil.MockWalletRepository{}
		itemRepo := &testutil.MockItemRepository{
			GetByIDErr: errors.New("db error"),
		}
		logger := zap.NewNop()
		svc := service.NewBankingService(&testutil.MockTxRunner{}, walletRepo, itemRepo, logger)
		handler := newHandler(svc, logger)

		c, recorder := newGinContext(http.MethodPost, "/api/v1/items/10/buyout", nil)
		c.Params = gin.Params{gin.Param{Key: "item_id", Value: "10"}}

		handler.Buyout(c)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		var resp map[string]string
		decodeJSON(t, recorder, &resp)
		assert.Equal(t, map[string]string{"error": "internal server error"}, resp)
	})
}
