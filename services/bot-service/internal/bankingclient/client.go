package bankingclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

var (
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrServiceUnavailable = errors.New("banking service unavailable")
	ErrItemNotFound       = errors.New("item not found")
)

type BankingServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

type WalletResponse struct {
	BotID   int64         `json:"bot_id"`
	Balance float64       `json:"balance"`
	Items   []ItemSummary `json:"items"`
}

type BuyoutResponse struct {
	NewBalance float64 `json:"new_balance"`
}

type ItemSummary struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	PurchasePrice float64 `json:"purchase_price"`
	AuctionID     int64   `json:"auction_id"`
}

func NewBankingServiceClient(baseURL string, logger *zap.Logger) *BankingServiceClient {
	return &BankingServiceClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

func (c *BankingServiceClient) GetWallet(ctx context.Context, botId int64) (*WalletResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/wallets/"+strconv.FormatInt(botId, 10), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to call banking service", zap.Int64("bot_id", botId), zap.Error(err))
		return nil, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrWalletNotFound
	}
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("unexpected response from banking service", zap.Int64("bot_id", botId), zap.Int("status_code", resp.StatusCode))
		return nil, ErrServiceUnavailable
	}

	var walletResp WalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&walletResp); err != nil {
		c.logger.Error("failed to decode response from banking service", zap.Int64("bot_id", botId), zap.Error(err))
		return nil, ErrServiceUnavailable
	}

	return &walletResp, nil
}

func (c *BankingServiceClient) Buyout(ctx context.Context, itemId int64) (newBalance float64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/banking/buyout/"+strconv.FormatInt(itemId, 10), nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to call banking service for buyout", zap.Int64("item_id", itemId), zap.Error(err))
		return 0, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, ErrItemNotFound
	}
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("unexpected response from banking service for buyout", zap.Int64("item_id", itemId), zap.Int("status_code", resp.StatusCode))
		return 0, ErrServiceUnavailable
	}

	var buyoutResp BuyoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&buyoutResp); err != nil {
		c.logger.Error("failed to decode response from banking service for buyout", zap.Int64("item_id", itemId), zap.Error(err))
		return 0, ErrServiceUnavailable
	}

	return buyoutResp.NewBalance, nil
}
