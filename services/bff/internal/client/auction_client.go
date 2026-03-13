package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"go.uber.org/zap"
)

type AuctionServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewAuctionServiceClient(baseURL string, logger *zap.Logger) *AuctionServiceClient {
	return &AuctionServiceClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

func (c *AuctionServiceClient) GetAuction(ctx context.Context) ([]*domain.AuctionView, error) {
	url := fmt.Sprintf("%s/api/auctions", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get auctions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get auctions: unexpected status %d", resp.StatusCode)
	}

	var auctions []*domain.AuctionView
	if err := json.NewDecoder(resp.Body).Decode(&auctions); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return auctions, nil
}
