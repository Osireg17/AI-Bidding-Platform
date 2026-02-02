package domain

import "errors"

// === CONTEXT ===
// Purpose: Sentinel errors for the auction domain.
// Service and handler layers match on these to decide HTTP status codes and log levels.
//
// === DATA / STATE ===
// Package-level sentinel errors — created once, compared by identity (errors.Is).

var (
	ErrAuctionNotFound     = errors.New("auction not found")
	ErrInvalidStatus       = errors.New("invalid auction status transition")
	ErrInvalidAuctionData  = errors.New("invalid auction data")
	ErrAuctionAlreadyEnded = errors.New("auction has already ended")
)
