package events

// === CONTEXT ===
// Purpose: Auction event payload structs and routing key constants.
// These define the contract between the auction-service (producer) and
// all consumers (bff, bot-service).
//
// === DEPENDENCIES ===
// None — pure data types and string constants.
//
// === DATA / STATE ===
// Payload structs are value types, created once per event, never mutated.

// Routing key constants for the auction.events topic exchange.
const (
	ExchangeName = "auction.events"
	ExchangeKind = "topic"

	RoutingKeyAuctionCreated    = "auction.created"
	RoutingKeyAuctionEndingSoon = "auction.ending_soon"
	RoutingKeyAuctionEnded      = "auction.ended"
)

// Event version — bump when payload shape changes.
const AuctionEventVersion = "1.0"

// AuctionCreatedPayload is published when a new auction is created and activated.
type AuctionCreatedPayload struct {
	AuctionID   int64   `json:"auction_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartPrice  float64 `json:"start_price"`
	StartTime   string  `json:"start_time"` // RFC 3339
	EndTime     string  `json:"end_time"`   // RFC 3339
}

// AuctionEndingSoonPayload is published when an active auction enters its ending-soon window.
type AuctionEndingSoonPayload struct {
	AuctionID int64  `json:"auction_id"`
	EndTime   string `json:"end_time"` // RFC 3339
}

// AuctionEndedPayload is published when an auction closes.
type AuctionEndedPayload struct {
	AuctionID   int64   `json:"auction_id"`
	WinnerBotID int64   `json:"winner_bot_id,omitempty"`
	WinningBid  float64 `json:"winning_bid,omitempty"`
	TotalBids   int     `json:"total_bids"`
	FinalStatus string  `json:"final_status"` // "sold" or "unsold"
}
