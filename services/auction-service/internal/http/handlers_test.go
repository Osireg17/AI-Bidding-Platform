package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/service"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/test"
	"github.com/gin-gonic/gin"
)

type auctionResponseDTO struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	StartPrice   float64 `json:"start_price"`
	CurrentPrice float64 `json:"current_price"`
	Status       string  `json:"status"`
	WinnerBotID  int64   `json:"winner_bot_id"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func newHandler(t *testing.T, repo *testutil.MockAuctionRepository, publisher *testutil.MockEventPublisher) *AuctionHandler {
	t.Helper()
	logger := testutil.NewTestLogger(t)
	svc := service.NewAuctionService(repo, publisher, nil, nil, logger)
	return NewAuctionHandler(svc, logger)
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

func TestHandleCreateAuction_Success(t *testing.T) {
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	payload := map[string]any{
		"title":        "New Auction",
		"description":  "Test auction",
		"start_price":  25.5,
		"duration_sec": 120,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	before := time.Now().UTC()
	c, recorder := newGinContext(http.MethodPost, "/api/auctions", body)
	handler.HandleCreateAuction(c)
	after := time.Now().UTC()

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response auctionResponseDTO
	decodeJSON(t, recorder, &response)

	if response.Title != payload["title"].(string) {
		t.Fatalf("expected title %q, got %q", payload["title"], response.Title)
	}
	if response.Description != payload["description"].(string) {
		t.Fatalf("expected description %q, got %q", payload["description"], response.Description)
	}
	if response.StartPrice != payload["start_price"].(float64) {
		t.Fatalf("expected start price %.2f, got %.2f", payload["start_price"], response.StartPrice)
	}
	if response.CurrentPrice != response.StartPrice {
		t.Fatalf("expected current price %.2f, got %.2f", response.StartPrice, response.CurrentPrice)
	}
	if response.Status != string(domain.StatusActive) {
		t.Fatalf("expected status %q, got %q", domain.StatusActive, response.Status)
	}

	startTime, err := time.Parse(time.RFC3339, response.StartTime)
	if err != nil {
		t.Fatalf("failed to parse start_time: %v", err)
	}
	endTime, err := time.Parse(time.RFC3339, response.EndTime)
	if err != nil {
		t.Fatalf("failed to parse end_time: %v", err)
	}
	if startTime.Before(before.Add(-time.Second)) || startTime.After(after.Add(time.Second)) {
		t.Fatalf("expected start_time within test window, got %v", startTime)
	}
	if !endTime.After(startTime) {
		t.Fatalf("expected end_time after start_time")
	}
	duration := endTime.Sub(startTime)
	if duration < 119*time.Second || duration > 121*time.Second {
		t.Fatalf("expected duration around 120s, got %v", duration)
	}

	if response.CreatedAt == "" || response.UpdatedAt == "" {
		t.Fatalf("expected created_at and updated_at to be set")
	}
}

func TestHandleCreateAuction_BindError(t *testing.T) {
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodPost, "/api/auctions", []byte("{invalid-json"))
	handler.HandleCreateAuction(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] == "" {
		t.Fatalf("expected error message in response")
	}
}

func TestHandleCreateAuction_InternalError(t *testing.T) {
	repo := &testutil.MockAuctionRepository{CreateErr: errors.New("db error")}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	payload := map[string]any{
		"title":        "New Auction",
		"description":  "Test auction",
		"start_price":  10.0,
		"duration_sec": 60,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	c, recorder := newGinContext(http.MethodPost, "/api/auctions", body)
	handler.HandleCreateAuction(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] != "internal server error" {
		t.Fatalf("expected internal server error, got %q", response["error"])
	}
}

func TestHandleGetAuction_Success(t *testing.T) {
	auction := testutil.CreateTestAuction(testutil.WithAuctionID(7), testutil.WithAuctionStatus(domain.StatusActive))
	repo := &testutil.MockAuctionRepository{GetByIDResult: auction}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions/7", nil)
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	handler.HandleGetAuction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response auctionResponseDTO
	decodeJSON(t, recorder, &response)
	if response.ID != 7 {
		t.Fatalf("expected id 7, got %d", response.ID)
	}
	if response.Title != auction.Title {
		t.Fatalf("expected title %q, got %q", auction.Title, response.Title)
	}
	if response.Status != string(auction.Status) {
		t.Fatalf("expected status %q, got %q", auction.Status, response.Status)
	}
}

func TestHandleGetAuction_InvalidID(t *testing.T) {
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	handler.HandleGetAuction(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] != "invalid auction id" {
		t.Fatalf("expected invalid auction id, got %q", response["error"])
	}
}

func TestHandleGetAuction_NotFound(t *testing.T) {
	repo := &testutil.MockAuctionRepository{GetByIDErr: domain.ErrAuctionNotFound}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions/99", nil)
	c.Params = gin.Params{{Key: "id", Value: "99"}}
	handler.HandleGetAuction(c)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] != "auction not found" {
		t.Fatalf("expected auction not found, got %q", response["error"])
	}
}

func TestHandleGetAuction_InternalError(t *testing.T) {
	repo := &testutil.MockAuctionRepository{GetByIDErr: errors.New("db error")}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions/10", nil)
	c.Params = gin.Params{{Key: "id", Value: "10"}}
	handler.HandleGetAuction(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] != "internal server error" {
		t.Fatalf("expected internal server error, got %q", response["error"])
	}
}

func TestHandleListAuctions_Success(t *testing.T) {
	auctions := []*domain.Auction{
		testutil.CreateTestAuction(testutil.WithAuctionID(1), testutil.WithAuctionTitle("Auction 1"), testutil.WithAuctionStatus(domain.StatusActive)),
		testutil.CreateTestAuction(testutil.WithAuctionID(2), testutil.WithAuctionTitle("Auction 2"), testutil.WithAuctionStatus(domain.StatusEndingSoon)),
	}
	repo := &testutil.MockAuctionRepository{ListResult: auctions}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions", nil)
	handler.HandleListAuctions(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response []auctionResponseDTO
	decodeJSON(t, recorder, &response)

	if len(response) != len(auctions) {
		t.Fatalf("expected %d auctions, got %d", len(auctions), len(response))
	}
	if response[0].ID != auctions[0].ID {
		t.Fatalf("expected first id %d, got %d", auctions[0].ID, response[0].ID)
	}
	if response[1].Title != auctions[1].Title {
		t.Fatalf("expected second title %q, got %q", auctions[1].Title, response[1].Title)
	}
}

func TestHandleListAuctions_InternalError(t *testing.T) {
	repo := &testutil.MockAuctionRepository{ListErr: errors.New("db error")}
	publisher := &testutil.MockEventPublisher{}
	handler := newHandler(t, repo, publisher)

	c, recorder := newGinContext(http.MethodGet, "/api/auctions", nil)
	handler.HandleListAuctions(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["error"] != "internal server error" {
		t.Fatalf("expected internal server error, got %q", response["error"])
	}
}
