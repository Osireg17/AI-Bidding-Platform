package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/service"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouterWithMocks(t *testing.T) (*gin.Engine, *testutil.MockBidRepository, *testutil.MockAuctionSnapshotRepository, *testutil.MockEventPublisher, *testutil.MockLockManager) {
	t.Helper()
	repo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{}
	publisher := &testutil.MockEventPublisher{}
	lockManager := &testutil.MockLockManager{}
	logger := testutil.NewTestLogger(t)
	svc := service.NewBidService(repo, snapshotRepo, lockManager, publisher, logger)
	handler := NewBidHandler(svc, logger)
	return NewRouter(handler, logger), repo, snapshotRepo, publisher, lockManager
}

func TestNewRouter_HealthEndpoint(t *testing.T) {
	router, _, _, _, _ := newRouterWithMocks(t)
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
	router, _, _, _, _ := newRouterWithMocks(t)
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
	router, _, _, _, _ := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Correlation-ID", "request-123")

	router.ServeHTTP(recorder, request)

	correlationID := recorder.Header().Get("X-Correlation-ID")
	if correlationID != "request-123" {
		t.Fatalf("expected correlation ID request-123, got %q", correlationID)
	}
}

func TestNewRouter_CreateBidRoute(t *testing.T) {
	router, _, _, _, _ := newRouterWithMocks(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/bids", nil)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	// 404 means the route was never registered.
	// Any other status confirms the route exists and reached the handler.
	require.NotEqual(t, http.StatusNotFound, recorder.Code, "expected route POST /api/bids to be registered")
	assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
}
