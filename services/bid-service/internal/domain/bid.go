package domain

import (
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type BidStatus string

const (
	StatusAccepted BidStatus = "accepted"
	StatusRejected BidStatus = "rejected"
)

type Bid struct {
	bun.BaseModel `bun:"table:bids,alias:b"`

	ID        int64     `bun:",pk,autoincrement"`
	AuctionID int64     `bun:",notnull"`
	BotID     int64     `bun:",notnull"`
	Amount    float64   `bun:",notnull"`
	Status    BidStatus `bun:",notnull"`
	Reason    string    `bun:",notnull,default:''"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// NewBid creates a new Bid in Accepted status.
func NewBid(auctionID, botID int64, amount float64) (*Bid, error) {
	if auctionID == 0 {
		return nil, fmt.Errorf("%w: auction ID is required", ErrInvalidBidData)
	}
	if botID == 0 {
		return nil, fmt.Errorf("%w: bot ID is required", ErrInvalidBidData)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", ErrInvalidBidData)
	}

	return &Bid{
		AuctionID: auctionID,
		BotID:     botID,
		Amount:    amount,
		Status:    StatusAccepted,
		Reason:    "",
		CreatedAt: time.Now().UTC(),
	}, nil
}
