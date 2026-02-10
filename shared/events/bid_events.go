package events

const (
	RoutingKeyBidPlaced   = "bid.placed"
	RoutingKeyBidRejected = "bid.rejected"
)

const BidEventVersion = "1.0"

type BidPlacedPayload struct {
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	BidAmount float64 `json:"bid_amount"`
	BidID     int64   `json:"bid_id"`
	Timestamp string  `json:"timestamp"` // RFC 3339
}

type BidRejectedPayload struct {
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	BidAmount float64 `json:"bid_amount"`
	BidID     int64   `json:"bid_id"`
	Reason    string  `json:"reason"`    // e.g., "bid too low", "auction ended", etc.
	Timestamp string  `json:"timestamp"` // RFC 3339
}
