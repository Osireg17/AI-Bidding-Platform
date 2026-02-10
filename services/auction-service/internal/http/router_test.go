package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/service"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newRouterWithMocks(t *testing.T) (*gin.Engine, *testutil.MockAuctionRepository, *testutil.MockEventPublisher) {
	t.Helper()
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)
	svc := service.NewAuctionService(repo, publisher, logger)
	handler := NewAuctionHandler(svc, logger)
	return NewRouter(handler, logger), repo, publisher
}

func TestNewRouter_HealthEndpoint(t *testing.T) {
	router, _, _ := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response map[string]string
	decodeJSON(t, recorder, &response)
	if response["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", response["status"])
	}
}

func TestNewRouter_HealthEndpointCorrelationIDGenerated(t *testing.T) {
	router, _, _ := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(recorder, request)

	correlationID := recorder.Header().Get("X-Correlation-ID")
	if correlationID == "" {
		t.Fatalf("expected X-Correlation-ID to be set")
	}
	if _, err := uuid.Parse(correlationID); err != nil {
		t.Fatalf("expected valid correlation ID, got %q", correlationID)
	}
}

func TestNewRouter_HealthEndpointCorrelationIDPreserved(t *testing.T) {
	router, _, _ := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Correlation-ID", "request-123")

	router.ServeHTTP(recorder, request)

	correlationID := recorder.Header().Get("X-Correlation-ID")
	if correlationID != "request-123" {
		t.Fatalf("expected correlation ID request-123, got %q", correlationID)
	}
}

func TestNewRouter_ListAuctionsRoute(t *testing.T) {
	router, repo, _ := newRouterWithMocks(t)
	repo.ListResult = []*domain.Auction{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/auctions", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response []map[string]any
	decodeJSON(t, recorder, &response)
	if len(response) != 0 {
		t.Fatalf("expected empty response, got %d items", len(response))
	}
	if repo.ListCalls != 1 {
		t.Fatalf("expected repo.List called once, got %d", repo.ListCalls)
	}
}

func TestNewRouter_CreateAuctionRoute(t *testing.T) {
	router, repo, publisher := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()

	payload := map[string]any{
		"title":        "Router Auction",
		"description":  "From router",
		"start_price":  12.5,
		"duration_sec": 60,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auctions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if repo.CreateCalls != 1 {
		t.Fatalf("expected repo.Create called once, got %d", repo.CreateCalls)
	}
	if publisher.PublishAuctionCreatedCalls != 1 {
		t.Fatalf("expected PublishAuctionCreated called once, got %d", publisher.PublishAuctionCreatedCalls)
	}
}

func TestNewRouter_GetAuctionRoute(t *testing.T) {
	router, repo, _ := newRouterWithMocks(t)
	repo.GetByIDResult = testutil.CreateTestAuction(
		testutil.WithAuctionID(42),
		testutil.WithAuctionTitle("Router Auction"),
		testutil.WithAuctionStatus(domain.StatusActive),
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/auctions/42", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response map[string]any
	decodeJSON(t, recorder, &response)
	if id, ok := response["id"].(float64); !ok || int64(id) != 42 {
		t.Fatalf("expected id 42, got %v", response["id"])
	}
}
