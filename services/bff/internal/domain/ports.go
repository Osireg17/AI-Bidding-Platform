package domain

import "github.com/Osireg17/AI-Bidding-Platform/shared/events"

// StateStore holds the current AuctionState in memory and applies event-driven updates.
// All Apply* methods are called by the MQ consumer as events arrive.
// GetState is called by REST and SSE handlers to read current store.
type StateStore interface {
	// GetState returns a snapshot copy of the current auction store.
	// Safe for concurrent reads.
	GetState() AuctionState

	// ApplyAuctionCreated resets the read model for a new auction.
	// Clears Bids and Winner, sets a fresh AuctionView.
	ApplyAuctionCreated(payload events.AuctionCreatedPayload)

	// ApplyAuctionEndingSoon updates the auction status to "ending_soon".
	ApplyAuctionEndingSoon(payload events.AuctionEndingSoonPayload)

	// ApplyAuctionEnded sets Winner and marks the auction as "closed".
	ApplyAuctionEnded(payload events.AuctionEndedPayload)

	// ApplyBidPlaced prepends a new BidView and updates CurrentPrice.
	ApplyBidPlaced(payload events.BidPlacedPayload)
}

// EventBroadcaster fans out SSE events to all connected browser clients.
// The MQ consumer calls Broadcast; SSE handlers call Subscribe.
type EventBroadcaster interface {
	// Broadcast marshals payload as JSON and sends it to all subscriber channels.
	// Non-blocking — drops the event for a subscriber whose channel is full.
	Broadcast(eventName string, payload any)

	// Subscribe registers a new SSE client and returns a channel that receives
	// SSEEvents and an unsubscribe function the handler must call on disconnect.
	Subscribe() (<-chan SSEEvent, func())
}
