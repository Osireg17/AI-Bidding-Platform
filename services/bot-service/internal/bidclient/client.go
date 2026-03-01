package bidclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

var (
	ErrAuctionNotFound    = errors.New("auction not found")
	ErrBidRejected        = errors.New("bid rejected")
	ErrServiceUnavailable = errors.New("bid service unavailable")
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

type placeBidRequest struct {
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	Amount    float64 `json:"amount"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (c *BidServiceClient) PlaceBid(ctx context.Context, auctionID, botID int64, amount float64) error {
	body, err := json.Marshal(placeBidRequest{
		AuctionID: auctionID,
		BotID:     botID,
		Amount:    amount,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/bids", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call bid service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	var errResp errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("bid service returned status %d", resp.StatusCode)
	}

	c.logger.Warn("bid service returned error",
		zap.Int("status", resp.StatusCode),
		zap.String("error", errResp.Error),
		zap.Int64("auction_id", auctionID),
		zap.Int64("bot_id", botID),
		zap.Float64("amount", amount),
	)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrAuctionNotFound
	case http.StatusConflict:
		return ErrBidRejected
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return fmt.Errorf("bid service error: %s", errResp.Error)
	}
}
