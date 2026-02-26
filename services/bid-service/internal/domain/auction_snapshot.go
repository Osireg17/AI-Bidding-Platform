package domain

import (
	"time"

	"github.com/uptrace/bun"
)

type AuctionStatus string

const (
	AuctionStatusActive     AuctionStatus = "active"
	AuctionStatusEndingSoon AuctionStatus = "ending_soon"
	AuctionStatusClosed     AuctionStatus = "closed"
)

type AuctionSnapshot struct {
	bun.BaseModel `bun:"table:auction_snapshots,alias:as"`

	AuctionID  int64         `bun:",pk"`
	Title      string        `bun:",notnull"`
	StartPrice float64       `bun:",notnull"`
	Status     AuctionStatus `bun:",notnull"`
	StartTime  time.Time     `bun:",notnull"`
	EndTime    time.Time     `bun:",notnull"`
	UpdatedAt  time.Time     `bun:",nullzero,notnull,default:current_timestamp"`
}

func (s *AuctionSnapshot) IsActive() bool {
	return s.Status == AuctionStatusActive || s.Status == AuctionStatusEndingSoon
}
