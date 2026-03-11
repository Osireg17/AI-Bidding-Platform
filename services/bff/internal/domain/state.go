package domain

import "time"

var BotNames = map[int64]string{
	1: "Aggressive Alice",
	2: "Sniper Steve",
	3: "Value Victor",
	4: "Chaos Charlie",
}

type AuctionState struct {
	Auction *AuctionView
	Bids    []BidView
	Winner  *WinnerView
}

type AuctionView struct {
	ID           int64
	Title        string
	Description  string
	StartPrice   float64
	CurrentPrice float64
	Status       string
	EndTime      time.Time
}

type BidView struct {
	BotName   string
	BotID     int64
	Amount    float64
	Timestamp time.Time
}

type WinnerView struct {
	BotName     string
	BotID       int64
	Amount      float64
	FinalStatus string // "sold" or "unsold"
}

// SSEEvent is the wire format written to an SSE response stream.
type SSEEvent struct {
	Name    string // e.g. "bid.placed", "auction.ended"
	Payload []byte // pre-marshalled JSON
}
