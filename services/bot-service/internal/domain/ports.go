package domain

import "context"

type BotBidRepository interface {
	Create(ctx context.Context, bid *BotBid) error

	GetByAuction(ctx context.Context, auctionID int64) ([]*BotBid, error)

	HasBidOnAuction(ctx context.Context, botID int64, auctionID int64) (bool, error)
}
