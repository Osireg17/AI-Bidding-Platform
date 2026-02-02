package domain

import "context"

// === CONTEXT ===
// Purpose: Port interfaces that define the boundaries of the domain layer.
// The domain depends on these interfaces; infrastructure packages implement them.
// This is the Ports & Adapters pattern — domain never imports infrastructure.
//
// === DEPENDENCIES ===
// context — for cancellation and deadlines on all operations.

// AuctionRepository defines persistence operations for Auction aggregates.
type AuctionRepository interface {
	// Create persists a new auction. Returns error if it already exists.
	Create(ctx context.Context, auction *Auction) error

	// GetByID retrieves a single auction by its ID.
	GetByID(ctx context.Context, id int64) (*Auction, error)

	// List retrieves all auctions, ordered by created_at descending.
	List(ctx context.Context) ([]*Auction, error)

	// Update persists changes to an existing auction.
	Update(ctx context.Context, auction *Auction) error

	// FindExpiredActive returns auctions that are active/ending_soon and past their end_time.
	FindExpiredActive(ctx context.Context) ([]*Auction, error)

	// FindEndingSoon returns active auctions within the ending-soon time threshold.
	FindEndingSoon(ctx context.Context, thresholdSeconds int) ([]*Auction, error)
}

// EventPublisher defines event publishing operations for auction lifecycle events.
type EventPublisher interface {
	// PublishAuctionCreated publishes an auction.created event.
	PublishAuctionCreated(ctx context.Context, auction *Auction) error

	// PublishAuctionEndingSoon publishes an auction.ending_soon event.
	PublishAuctionEndingSoon(ctx context.Context, auction *Auction) error

	// PublishAuctionEnded publishes an auction.ended event.
	PublishAuctionEnded(ctx context.Context, auction *Auction) error

	// Close cleans up publisher resources.
	Close() error
}
