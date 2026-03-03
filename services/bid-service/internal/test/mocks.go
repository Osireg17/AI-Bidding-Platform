package test

import (
	"context"
	"sync"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
)

// --- MockBidRepository ---

type MockBidRepository struct {
	mu sync.Mutex

	CreateCalls int
	CreateArgs  []CreateBidArgs
	CreateErr   error

	GetHighestBidCalls  int
	GetHighestBidArgs   []GetHighestBidArgs
	GetHighestBidResult float64
	GetHighestBidErr    error

	ListByAuctionCalls  int
	ListByAuctionArgs   []ListByAuctionArgs
	ListByAuctionResult []*domain.Bid
	ListByAuctionErr    error
}

func (m *MockBidRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls = 0
	m.CreateArgs = nil
	m.CreateErr = nil

	m.GetHighestBidCalls = 0
	m.GetHighestBidArgs = nil
	m.GetHighestBidResult = 0
	m.GetHighestBidErr = nil

	m.ListByAuctionCalls = 0
	m.ListByAuctionArgs = nil
	m.ListByAuctionResult = nil
	m.ListByAuctionErr = nil
}

type CreateBidArgs struct {
	Ctx context.Context
	Bid *domain.Bid
}

func (m *MockBidRepository) Create(ctx context.Context, bid *domain.Bid) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalls++
	m.CreateArgs = append(m.CreateArgs, CreateBidArgs{Ctx: ctx, Bid: bid})
	return m.CreateErr
}

type GetHighestBidArgs struct {
	Ctx       context.Context
	AuctionID int64
}

func (m *MockBidRepository) GetHighestBid(ctx context.Context, auctionID int64) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetHighestBidCalls++
	m.GetHighestBidArgs = append(m.GetHighestBidArgs, GetHighestBidArgs{Ctx: ctx, AuctionID: auctionID})
	return m.GetHighestBidResult, m.GetHighestBidErr
}

type ListByAuctionArgs struct {
	Ctx       context.Context
	AuctionID int64
}

func (m *MockBidRepository) ListByAuction(ctx context.Context, auctionID int64) ([]*domain.Bid, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListByAuctionCalls++
	m.ListByAuctionArgs = append(m.ListByAuctionArgs, ListByAuctionArgs{Ctx: ctx, AuctionID: auctionID})
	return m.ListByAuctionResult, m.ListByAuctionErr
}

type GetWinnerArgs struct {
	Ctx       context.Context
	AuctionID int64
}

func (m *MockBidRepository) GetWinner(context.Context, int64) (int64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return 0, 0, nil
}

// --- MockAuctionSnapshotRepository ---

type MockAuctionSnapshotRepository struct {
	mu sync.Mutex

	UpsertCalls int
	UpsertArgs  []UpsertArgs
	UpsertErr   error

	GetByIDCalls  int
	GetByIDArgs   []GetSnapshotByIDArgs
	GetByIDResult *domain.AuctionSnapshot
	GetByIDErr    error

	UpdateStatusCalls int
	UpdateStatusArgs  []UpdateStatusArgs
	UpdateStatusErr   error
}

func (m *MockAuctionSnapshotRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpsertCalls = 0
	m.UpsertArgs = nil
	m.UpsertErr = nil

	m.GetByIDCalls = 0
	m.GetByIDArgs = nil
	m.GetByIDResult = nil
	m.GetByIDErr = nil

	m.UpdateStatusCalls = 0
	m.UpdateStatusArgs = nil
	m.UpdateStatusErr = nil
}

type UpsertArgs struct {
	Ctx      context.Context
	Snapshot *domain.AuctionSnapshot
}

func (m *MockAuctionSnapshotRepository) Upsert(ctx context.Context, snapshot *domain.AuctionSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpsertCalls++
	m.UpsertArgs = append(m.UpsertArgs, UpsertArgs{Ctx: ctx, Snapshot: snapshot})
	return m.UpsertErr
}

type GetSnapshotByIDArgs struct {
	Ctx       context.Context
	AuctionID int64
}

func (m *MockAuctionSnapshotRepository) GetByID(ctx context.Context, auctionID int64) (*domain.AuctionSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetByIDCalls++
	m.GetByIDArgs = append(m.GetByIDArgs, GetSnapshotByIDArgs{Ctx: ctx, AuctionID: auctionID})
	return m.GetByIDResult, m.GetByIDErr
}

type UpdateStatusArgs struct {
	Ctx       context.Context
	AuctionID int64
	Status    domain.AuctionStatus
}

func (m *MockAuctionSnapshotRepository) UpdateStatus(ctx context.Context, auctionID int64, status domain.AuctionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateStatusCalls++
	m.UpdateStatusArgs = append(m.UpdateStatusArgs, UpdateStatusArgs{Ctx: ctx, AuctionID: auctionID, Status: status})
	return m.UpdateStatusErr
}

// --- MockEventPublisher ---

type MockEventPublisher struct {
	mu sync.Mutex

	PublishBidPlacedCalls int
	PublishBidPlacedArgs  []PublishBidPlacedArgs
	PublishBidPlacedErr   error

	PublishBidRejectedCalls int
	PublishBidRejectedArgs  []PublishBidRejectedArgs
	PublishBidRejectedErr   error

	CloseCalls int
	CloseErr   error
}

func (m *MockEventPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishBidPlacedCalls = 0
	m.PublishBidPlacedArgs = nil
	m.PublishBidPlacedErr = nil

	m.PublishBidRejectedCalls = 0
	m.PublishBidRejectedArgs = nil
	m.PublishBidRejectedErr = nil

	m.CloseCalls = 0
	m.CloseErr = nil
}

type PublishBidPlacedArgs struct {
	Ctx context.Context
	Bid *domain.Bid
}

func (m *MockEventPublisher) PublishBidPlaced(ctx context.Context, bid *domain.Bid) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishBidPlacedCalls++
	m.PublishBidPlacedArgs = append(m.PublishBidPlacedArgs, PublishBidPlacedArgs{Ctx: ctx, Bid: bid})
	return m.PublishBidPlacedErr
}

type PublishBidRejectedArgs struct {
	Ctx    context.Context
	Bid    *domain.Bid
	Reason string
}

func (m *MockEventPublisher) PublishBidRejected(ctx context.Context, bid *domain.Bid, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishBidRejectedCalls++
	m.PublishBidRejectedArgs = append(m.PublishBidRejectedArgs, PublishBidRejectedArgs{Ctx: ctx, Bid: bid, Reason: reason})
	return m.PublishBidRejectedErr
}

func (m *MockEventPublisher) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalls++
	return m.CloseErr
}

// --- MockLockManager ---

type MockLockManager struct {
	mu sync.Mutex

	AcquireLockCalls int
	AcquireLockArgs  []AcquireLockArgs
	AcquireLockErr   error

	ReleaseLockCalls int
	ReleaseLockArgs  []ReleaseLockArgs
	ReleaseLockErr   error
}

func (m *MockLockManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.AcquireLockCalls = 0
	m.AcquireLockArgs = nil
	m.AcquireLockErr = nil

	m.ReleaseLockCalls = 0
	m.ReleaseLockArgs = nil
	m.ReleaseLockErr = nil
}

type AcquireLockArgs struct {
	Ctx       context.Context
	AuctionID int64
	TTL       time.Duration
}

func (m *MockLockManager) AcquireLock(ctx context.Context, auctionID int64, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AcquireLockCalls++
	m.AcquireLockArgs = append(m.AcquireLockArgs, AcquireLockArgs{Ctx: ctx, AuctionID: auctionID, TTL: ttl})
	return m.AcquireLockErr
}

type ReleaseLockArgs struct {
	Ctx       context.Context
	AuctionID int64
}

func (m *MockLockManager) ReleaseLock(ctx context.Context, auctionID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReleaseLockCalls++
	m.ReleaseLockArgs = append(m.ReleaseLockArgs, ReleaseLockArgs{Ctx: ctx, AuctionID: auctionID})
	return m.ReleaseLockErr
}

// Compile-time interface guards.
var _ domain.BidRepository = (*MockBidRepository)(nil)
var _ domain.AuctionSnapshotRepository = (*MockAuctionSnapshotRepository)(nil)
var _ domain.EventPublisher = (*MockEventPublisher)(nil)
var _ domain.LockManager = (*MockLockManager)(nil)
