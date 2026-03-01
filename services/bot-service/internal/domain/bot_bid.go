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

type BotBid struct {
	bun.BaseModel `bun:"table:bot_bids,alias:bb"`

	ID        int64     `bun:",pk,autoincrement"`
	BotID     int64     `bun:",notnull"`
	AuctionID int64     `bun:",notnull"`
	Amount    float64   `bun:",notnull"`
	Status    BidStatus `bun:",notnull"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

func NewBotBid(botID, auctionID int64, amount float64) (*BotBid, error) {
	if botID == 0 {
		return nil, fmt.Errorf(
			"%w: bot ID is required", ErrInvalidBotBidData)
	}
	if auctionID == 0 {
		return nil, fmt.Errorf(
			"%w: auction ID is required", ErrInvalidBotBidData)
	}
	if amount <= 0 {
		return nil, fmt.Errorf(
			"%w: amount must be positive", ErrInvalidBotBidData)
	}

	return &BotBid{
		BotID:     botID,
		AuctionID: auctionID,
		Amount:    amount,
		Status:    StatusAccepted,
		CreatedAt: time.Now().UTC(),
	}, nil
}
