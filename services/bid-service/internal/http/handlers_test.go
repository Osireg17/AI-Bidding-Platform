package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/service"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bidResponseDTO struct {
	ID        int64   `json:"id"`
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type errorResponseDTO struct {
	Error string `json:"error"`
}

func newHandler(t *testing.T, bidRepo *testutil.MockBidRepository, publisher *testutil.MockEventPublisher, snapshotRepo *testutil.MockAuctionSnapshotRepository, lockMgr *testutil.MockLockManager) *BidHandler {
	t.Helper()
	logger := testutil.NewTestLogger(t)
	svc := service.NewBidService(bidRepo, snapshotRepo, lockMgr, publisher, logger)
	return NewBidHandler(svc, logger)
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
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(target))
}

func activeSnapshotFor(auctionID int64, startPrice float64) *testutil.MockAuctionSnapshotRepository {
	return &testutil.MockAuctionSnapshotRepository{
		GetByIDResult: testutil.CreateTestSnapshot(
			testutil.WithSnapshotAuctionID(auctionID),
			testutil.WithSnapshotStartPrice(startPrice),
			testutil.WithSnapshotStatus(domain.AuctionStatusActive),
		),
	}
}

// TestHandlePlaceBid_Success: valid request, auction exists, amount beats start price.
func TestHandlePlaceBid_Success(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
	snapshotRepo := activeSnapshotFor(1, 10.0)
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 1, BotID: 2, Amount: 100.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp bidResponseDTO
	decodeJSON(t, recorder, &resp)

	assert.Equal(t, int64(1), resp.AuctionID)
	assert.Equal(t, int64(2), resp.BotID)
	assert.Equal(t, 100.0, resp.Amount)
	assert.Equal(t, "accepted", resp.Status)
	assert.NotEmpty(t, resp.CreatedAt)
}

// TestHandlePlaceBid_MalformedJSON: body is not valid JSON.
func TestHandlePlaceBid_MalformedJSON(t *testing.T) {
	handler := newHandler(t, &testutil.MockBidRepository{}, &testutil.MockEventPublisher{}, &testutil.MockAuctionSnapshotRepository{}, &testutil.MockLockManager{})

	c, recorder := newGinContext("POST", "/api/bids", []byte("not-json"))

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestHandlePlaceBid_AuctionNotFound: snapshot repo returns nil.
func TestHandlePlaceBid_AuctionNotFound(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: nil}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 99, BotID: 1, Amount: 50.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)

	var resp errorResponseDTO
	decodeJSON(t, recorder, &resp)
	assert.NotEmpty(t, resp.Error)
}

// TestHandlePlaceBid_AuctionNotActive: auction exists but is closed.
func TestHandlePlaceBid_AuctionNotActive(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{
		GetByIDResult: testutil.CreateTestSnapshot(
			testutil.WithSnapshotAuctionID(1),
			testutil.WithSnapshotStatus(domain.AuctionStatusClosed),
		),
	}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 1, BotID: 1, Amount: 50.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusConflict, recorder.Code)

	var resp errorResponseDTO
	decodeJSON(t, recorder, &resp)
	assert.NotEmpty(t, resp.Error)
}

// TestHandlePlaceBid_BidTooLow: amount does not beat the current highest bid.
func TestHandlePlaceBid_BidTooLow(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 200.0}
	snapshotRepo := activeSnapshotFor(1, 10.0)
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 1, BotID: 1, Amount: 50.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusConflict, recorder.Code)

	var resp errorResponseDTO
	decodeJSON(t, recorder, &resp)
	assert.NotEmpty(t, resp.Error)
}

// TestHandlePlaceBid_LockAcquisitionFailed: Redis lock unavailable.
func TestHandlePlaceBid_LockAcquisitionFailed(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := activeSnapshotFor(1, 10.0)
	lockMgr := &testutil.MockLockManager{AcquireLockErr: domain.ErrLockAcquisitionFailed}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 1, BotID: 1, Amount: 50.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	var resp errorResponseDTO
	decodeJSON(t, recorder, &resp)
	assert.NotEmpty(t, resp.Error)
}

// TestHandlePlaceBid_InternalError: unexpected repo error maps to 500.
func TestHandlePlaceBid_InternalError(t *testing.T) {
	bidRepo := &testutil.MockBidRepository{GetHighestBidErr: errors.New("db down")}
	snapshotRepo := activeSnapshotFor(1, 10.0)
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	handler := newHandler(t, bidRepo, publisher, snapshotRepo, lockMgr)

	reqBody := createBidRequest{AuctionID: 1, BotID: 1, Amount: 50.0}
	bodyBytes, _ := json.Marshal(reqBody)
	c, recorder := newGinContext("POST", "/api/bids", bodyBytes)

	handler.HandlePlaceBid(c)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var resp errorResponseDTO
	decodeJSON(t, recorder, &resp)
	assert.Equal(t, "internal server error", resp.Error)
}
