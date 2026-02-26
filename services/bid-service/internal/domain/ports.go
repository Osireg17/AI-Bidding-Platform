package domain

import (
	"context"
	"time"
)

type BidRepository interface {
	Create(ctx context.Context, bid *Bid) error

	GetHighestBid(ctx context.Context, auctionID int64) (float64, error)

	ListByAuction(ctx context.Context, auctionID int64) ([]*Bid, error)
}

type AuctionSnapshotRepository interface {
	Upsert(ctx context.Context, snapshot *AuctionSnapshot) error

	GetByID(ctx context.Context, auctionID int64) (*AuctionSnapshot, error)

	UpdateStatus(ctx context.Context, auctionID int64, status AuctionStatus) error
}

type EventPublisher interface {
	PublishBidPlaced(ctx context.Context, bid *Bid) error

	PublishBidRejected(ctx context.Context, bid *Bid, reason string) error

	Close() error
}

type LockManager interface {
	AcquireLock(ctx context.Context, auctionID int64, ttl time.Duration) error

	ReleaseLock(ctx context.Context, auctionID int64) error
}
