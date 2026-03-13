package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"go.uber.org/zap"
)

type BidServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewBidServiceClient(baseURL string, logger *zap.Logger) *BidServiceClient {
	return &BidServiceClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

func (c *BidServiceClient) GetHighestBid(ctx context.Context, auctionID int64) (int64, float64, error) {
	url := fmt.Sprintf("%s/api/bids/highest?auction_id=%d", c.baseURL, auctionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("get highest bid: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("get highest bid: unexpected status %d", resp.StatusCode)
	}

	var bid domain.BidView
	if err := json.NewDecoder(resp.Body).Decode(&bid); err != nil {
		return 0, 0, fmt.Errorf("decode response: %w", err)
	}

	return bid.BotID, bid.Amount, nil
}
