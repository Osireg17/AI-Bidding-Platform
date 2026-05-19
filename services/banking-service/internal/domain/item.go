package domain

import (
	"time"

	"github.com/uptrace/bun"
)

type Item struct {
	bun.BaseModel `bun:"table:items"`
	ID            int64     `bun:",pk,autoincrement"`
	BotID         int64     `bun:",notnull"`
	AuctionID     int64     `bun:",notnull"`
	Title         string    `bun:",notnull"`
	PurchasePrice float64   `bun:",notnull"`
	AcquiredAt    time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

func NewItem(botID, auctionID int64, title string, purchasePrice float64) (*Item, error) {
	//check if all fields non-zero/non-empty
	if botID == 0 {
		return nil, ErrInvalidBotID
	}
	if auctionID == 0 {
		return nil, ErrInvalidAuctionID
	}
	if title == "" {
		return nil, ErrInvalidTitle
	}
	if purchasePrice <= 0 {
		return nil, ErrInvalidPurchasePrice
	}

	// set AcquiredAt to current time
	return &Item{
		BotID:         botID,
		AuctionID:     auctionID,
		Title:         title,
		PurchasePrice: purchasePrice,
		AcquiredAt:    time.Now(),
	}, nil
}
