package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(
	bidRepo *testutil.MockBidRepository,
	snapshotRepo *testutil.MockAuctionSnapshotRepository,
	lockMgr *testutil.MockLockManager,
	publisher *testutil.MockEventPublisher,
	t *testing.T,
) *BidService {
	t.Helper()
	return NewBidService(bidRepo, snapshotRepo, lockMgr, publisher, testutil.NewTestLogger(t))
}

// TestPlaceBid_HappyPath_FirstBid: no prior bids exist so amount must beat start_price.
func TestPlaceBid_HappyPath_FirstBid(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	bid, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.NoError(t, err)
	require.NotNil(t, bid)

	expected := &domain.Bid{AuctionID: 1, BotID: 100, Amount: 20.0, Status: domain.StatusAccepted}
	bid.CreatedAt = time.Time{}
	assert.Equal(t, expected, bid)

	assert.Equal(t, 1, bidRepo.CreateCalls)
	assert.Same(t, bid, bidRepo.CreateArgs[0].Bid)

	assert.Equal(t, 1, publisher.PublishBidPlacedCalls)
	assert.Same(t, bid, publisher.PublishBidPlacedArgs[0].Bid)
	assert.Equal(t, 0, publisher.PublishBidRejectedCalls)

	assert.Equal(t, 1, lockMgr.AcquireLockCalls)
	assert.Equal(t, int64(1), lockMgr.AcquireLockArgs[0].AuctionID)
	assert.Equal(t, 1, lockMgr.ReleaseLockCalls)
}

// TestPlaceBid_HappyPath_OutbidsExisting: a prior bid exists and the new amount beats it.
func TestPlaceBid_HappyPath_OutbidsExisting(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 50.0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	bid, err := svc.PlaceBid(ctx, 1, 100, 75.0)
	require.NoError(t, err)
	require.NotNil(t, bid)

	expected := &domain.Bid{AuctionID: 1, BotID: 100, Amount: 75.0, Status: domain.StatusAccepted}
	bid.CreatedAt = time.Time{}
	assert.Equal(t, expected, bid)

	assert.Equal(t, 1, bidRepo.CreateCalls)
	assert.Same(t, bid, bidRepo.CreateArgs[0].Bid)

	assert.Equal(t, 1, publisher.PublishBidPlacedCalls)
	assert.Same(t, bid, publisher.PublishBidPlacedArgs[0].Bid)
}

// TestPlaceBid_HappyPath_EndingSoon: ending_soon auctions are still biddable per domain.IsActive().
func TestPlaceBid_HappyPath_EndingSoon(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusEndingSoon),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	bid, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.NoError(t, err)
	require.NotNil(t, bid)

	expected := &domain.Bid{AuctionID: 1, BotID: 100, Amount: 20.0, Status: domain.StatusAccepted}
	bid.CreatedAt = time.Time{}
	assert.Equal(t, expected, bid)

	assert.Equal(t, 1, bidRepo.CreateCalls)
	assert.Equal(t, 1, publisher.PublishBidPlacedCalls)
	assert.Equal(t, 0, publisher.PublishBidRejectedCalls)
}

// TestPlaceBid_LockAcquisitionFailed: AcquireLock fails — no further calls made.
func TestPlaceBid_LockAcquisitionFailed(t *testing.T) {
	ctx := context.Background()
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{}
	lockMgr := &testutil.MockLockManager{AcquireLockErr: domain.ErrLockAcquisitionFailed}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, domain.ErrLockAcquisitionFailed)

	assert.Equal(t, 0, snapshotRepo.GetByIDCalls)
	assert.Equal(t, 0, bidRepo.CreateCalls)
	// defer never ran because the lock was never acquired
	assert.Equal(t, 0, lockMgr.ReleaseLockCalls)
}

// TestPlaceBid_AuctionNotFound: GetByID returns nil snapshot with no error.
func TestPlaceBid_AuctionNotFound(t *testing.T) {
	ctx := context.Background()
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: nil, GetByIDErr: nil}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, domain.ErrAuctionNotFound)

	assert.Equal(t, 0, bidRepo.CreateCalls)
	// defer still fires even on early return
	assert.Equal(t, 1, lockMgr.ReleaseLockCalls)
}

// TestPlaceBid_SnapshotRepo_Error: GetByID returns a hard repo error.
func TestPlaceBid_SnapshotRepo_Error(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db connection lost")
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDErr: dbErr}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, dbErr)

	assert.Equal(t, 0, bidRepo.CreateCalls)
}

// TestPlaceBid_AuctionNotActive: snapshot exists but status is closed.
func TestPlaceBid_AuctionNotActive(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStatus(domain.AuctionStatusClosed),
	)
	bidRepo := &testutil.MockBidRepository{}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, domain.ErrAuctionNotActive)

	// validation failed before we ever touch the bid repo
	assert.Equal(t, 0, bidRepo.GetHighestBidCalls)
	assert.Equal(t, 0, bidRepo.CreateCalls)
}

// TestPlaceBid_BidTooLow_NoExistingBids: amount does not beat start_price.
func TestPlaceBid_BidTooLow_NoExistingBids(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(50.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	// equal to start_price — must be strictly greater
	_, err := svc.PlaceBid(ctx, 1, 100, 50.0)
	require.ErrorIs(t, err, domain.ErrBidTooLow)

	assert.Equal(t, 0, bidRepo.CreateCalls)

	require.Equal(t, 1, publisher.PublishBidRejectedCalls)
	rejectedArgs := publisher.PublishBidRejectedArgs[0]
	expectedRejected := &domain.Bid{AuctionID: 1, BotID: 100, Amount: 50.0}
	assert.Equal(t, expectedRejected, rejectedArgs.Bid)
	assert.Equal(t, "bid too low", rejectedArgs.Reason)

	assert.Equal(t, 0, publisher.PublishBidPlacedCalls)
}

// TestPlaceBid_BidTooLow_ExistingBids: amount does not beat the current highest bid.
func TestPlaceBid_BidTooLow_ExistingBids(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 100.0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	// equal to highest bid — must be strictly greater
	_, err := svc.PlaceBid(ctx, 1, 200, 100.0)
	require.ErrorIs(t, err, domain.ErrBidTooLow)

	require.Equal(t, 1, publisher.PublishBidRejectedCalls)
	rejectedArgs := publisher.PublishBidRejectedArgs[0]
	expectedRejected := &domain.Bid{AuctionID: 1, BotID: 200, Amount: 100.0}
	assert.Equal(t, expectedRejected, rejectedArgs.Bid)
	assert.Equal(t, "bid too low", rejectedArgs.Reason)
}

// TestPlaceBid_GetHighestBid_RepoError: GetHighestBid fails — no bid created.
func TestPlaceBid_GetHighestBid_RepoError(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	dbErr := errors.New("db timeout")
	bidRepo := &testutil.MockBidRepository{GetHighestBidErr: dbErr}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, dbErr)

	assert.Equal(t, 0, bidRepo.CreateCalls)
	assert.Equal(t, 0, publisher.PublishBidPlacedCalls)
}

// TestPlaceBid_Create_RepoError: bid passes validation but repo.Create fails.
func TestPlaceBid_Create_RepoError(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	dbErr := errors.New("insert failed")
	bidRepo := &testutil.MockBidRepository{
		GetHighestBidResult: 0,
		CreateErr:           dbErr,
	}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	_, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.ErrorIs(t, err, dbErr)

	assert.Equal(t, 0, publisher.PublishBidPlacedCalls)
}

// TestPlaceBid_PublishError_DoesNotFail: publish failure is logged but PlaceBid still succeeds.
func TestPlaceBid_PublishError_DoesNotFail(t *testing.T) {
	ctx := context.Background()
	snapshot := testutil.CreateTestSnapshot(
		testutil.WithSnapshotAuctionID(1),
		testutil.WithSnapshotStartPrice(10.0),
		testutil.WithSnapshotStatus(domain.AuctionStatusActive),
	)
	bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
	snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
	lockMgr := &testutil.MockLockManager{}
	publisher := &testutil.MockEventPublisher{PublishBidPlacedErr: errors.New("rabbitmq down")}

	svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

	bid, err := svc.PlaceBid(ctx, 1, 100, 20.0)
	require.NoError(t, err)
	require.NotNil(t, bid)

	assert.Equal(t, domain.StatusAccepted, bid.Status)
	assert.Equal(t, 1, bidRepo.CreateCalls)
	assert.Equal(t, 1, publisher.PublishBidPlacedCalls)
}

// TestPlaceBid_LockReleasedOnEveryPath: defer releases lock regardless of which step fails.
func TestPlaceBid_LockReleasedOnEveryPath(t *testing.T) {
	cases := []struct {
		name         string
		snapshotRepo *testutil.MockAuctionSnapshotRepository
		bidRepo      *testutil.MockBidRepository
	}{
		{
			name:         "snapshot not found",
			snapshotRepo: &testutil.MockAuctionSnapshotRepository{GetByIDResult: nil},
			bidRepo:      &testutil.MockBidRepository{},
		},
		{
			name: "auction not active",
			snapshotRepo: &testutil.MockAuctionSnapshotRepository{
				GetByIDResult: testutil.CreateTestSnapshot(testutil.WithSnapshotStatus(domain.AuctionStatusClosed)),
			},
			bidRepo: &testutil.MockBidRepository{},
		},
		{
			name: "bid too low",
			snapshotRepo: &testutil.MockAuctionSnapshotRepository{
				GetByIDResult: testutil.CreateTestSnapshot(testutil.WithSnapshotStartPrice(100.0)),
			},
			bidRepo: &testutil.MockBidRepository{GetHighestBidResult: 0},
		},
		{
			name: "create repo error",
			snapshotRepo: &testutil.MockAuctionSnapshotRepository{
				GetByIDResult: testutil.CreateTestSnapshot(testutil.WithSnapshotStartPrice(10.0)),
			},
			bidRepo: &testutil.MockBidRepository{
				GetHighestBidResult: 0,
				CreateErr:           errors.New("insert failed"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lockMgr := &testutil.MockLockManager{}
			publisher := &testutil.MockEventPublisher{}
			svc := newTestService(tc.bidRepo, tc.snapshotRepo, lockMgr, publisher, t)

			_, _ = svc.PlaceBid(context.Background(), 1, 100, 5.0)

			assert.Equal(t, 1, lockMgr.ReleaseLockCalls)
		})
	}
}
