package events

const (
	ExchangeName = "auction.events"
	ExchangeKind = "topic"

	RoutingKeyAuctionCreated    = "auction.created"
	RoutingKeyAuctionEndingSoon = "auction.ending_soon"
	RoutingKeyAuctionEnded      = "auction.ended"
)

const AuctionEventVersion = "1.0"

type AuctionCreatedPayload struct {
	AuctionID   int64   `json:"auction_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartPrice  float64 `json:"start_price"`
	StartTime   string  `json:"start_time"` // RFC 3339
	EndTime     string  `json:"end_time"`   // RFC 3339
}

type AuctionEndingSoonPayload struct {
	AuctionID int64  `json:"auction_id"`
	EndTime   string `json:"end_time"` // RFC 3339
}

type AuctionEndedPayload struct {
	AuctionID   int64   `json:"auction_id"`
	WinnerBotID int64   `json:"winner_bot_id,omitempty"`
	WinningBid  float64 `json:"winning_bid,omitempty"`
	TotalBids   int     `json:"total_bids"`
	FinalStatus string  `json:"final_status"` // "sold" or "unsold"
	Title       string  `json:"title"`
}
