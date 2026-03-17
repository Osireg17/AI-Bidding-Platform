package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

type auctionResponse struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	StartPrice   float64 `json:"start_price"`
	CurrentPrice float64 `json:"current_price"`
	Status       string  `json:"status"`
	EndTime      string  `json:"end_time"`
}

func (c *AuctionServiceClient) GetActiveAuction(ctx context.Context) (*domain.AuctionView, error) {
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

	var auctions []auctionResponse
	if err := json.NewDecoder(resp.Body).Decode(&auctions); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.logger.Info("fetched auctions from auction-service", zap.Int("count", len(auctions)))

	for _, a := range auctions {
		if a.Status == "active" || a.Status == "ending_soon" {
			endTime, err := time.Parse(time.RFC3339, a.EndTime)
			if err != nil {
				return nil, fmt.Errorf("parse end_time for auction %d: %w", a.ID, err)
			}
			return &domain.AuctionView{
				ID:           a.ID,
				Title:        a.Title,
				Description:  a.Description,
				StartPrice:   a.StartPrice,
				CurrentPrice: a.CurrentPrice,
				Status:       a.Status,
				EndTime:      endTime,
			}, nil
		}
	}

	return nil, nil
}
