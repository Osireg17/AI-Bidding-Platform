package test

import (
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func NewTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zaptest.NewLogger(t)
}

// BidOption is a functional option for building test Bid values.
type BidOption func(*domain.Bid)

func CreateTestBid(opts ...BidOption) *domain.Bid {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	bid := &domain.Bid{
		ID:        1,
		AuctionID: 10,
		BotID:     100,
		Amount:    50.0,
		Status:    domain.StatusAccepted,
		Reason:    "",
		CreatedAt: now,
	}
	for _, opt := range opts {
		opt(bid)
	}
	return bid
}

func WithBidID(id int64) BidOption {
	return func(b *domain.Bid) { b.ID = id }
}

func WithBidAuctionID(id int64) BidOption {
	return func(b *domain.Bid) { b.AuctionID = id }
}

func WithBidBotID(id int64) BidOption {
	return func(b *domain.Bid) { b.BotID = id }
}

func WithBidAmount(amount float64) BidOption {
	return func(b *domain.Bid) { b.Amount = amount }
}

func WithBidStatus(status domain.BidStatus) BidOption {
	return func(b *domain.Bid) { b.Status = status }
}

func WithBidReason(reason string) BidOption {
	return func(b *domain.Bid) { b.Reason = reason }
}

// SnapshotOption is a functional option for building test AuctionSnapshot values.
type SnapshotOption func(*domain.AuctionSnapshot)

func CreateTestSnapshot(opts ...SnapshotOption) *domain.AuctionSnapshot {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	snapshot := &domain.AuctionSnapshot{
		AuctionID:  10,
		Title:      "Test Auction",
		StartPrice: 10.0,
		Status:     domain.AuctionStatusActive,
		StartTime:  now,
		EndTime:    now.Add(1 * time.Hour),
		UpdatedAt:  now,
	}
	for _, opt := range opts {
		opt(snapshot)
	}
	return snapshot
}

func WithSnapshotAuctionID(id int64) SnapshotOption {
	return func(s *domain.AuctionSnapshot) { s.AuctionID = id }
}

func WithSnapshotStatus(status domain.AuctionStatus) SnapshotOption {
	return func(s *domain.AuctionSnapshot) { s.Status = status }
}

func WithSnapshotStartPrice(price float64) SnapshotOption {
	return func(s *domain.AuctionSnapshot) { s.StartPrice = price }
}
