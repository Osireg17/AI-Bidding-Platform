package domain

import "context"

type AuctionRepository interface {
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
