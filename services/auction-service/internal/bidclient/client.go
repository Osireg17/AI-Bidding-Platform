package bidclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// BidServiceClient is a thin HTTP client for querying the bid-service.
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

type highestBidResponse struct {
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	Amount    float64 `json:"amount"`
}

// GetWinner calls GET /api/bids/highest?auction_id=<id> and returns the
// winning botID and amount. Returns (0, 0, nil) when no bids have been placed.
func (c *BidServiceClient) GetWinner(ctx context.Context, auctionID int64) (botID int64, amount float64, err error) {
	url := fmt.Sprintf("%s/api/bids/highest?auction_id=%d", c.baseURL, auctionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("get winner: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("get winner: unexpected status %d", resp.StatusCode)
	}

	var body highestBidResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, 0, fmt.Errorf("decode response: %w", err)
	}

	return body.BotID, body.Amount, nil
}
