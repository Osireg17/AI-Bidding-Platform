package domain

import "errors"

var (
	ErrAuctionNotFound     = errors.New("auction not found")
	ErrInvalidStatus       = errors.New("invalid auction status transition")
	ErrInvalidAuctionData  = errors.New("invalid auction data")
	ErrAuctionAlreadyEnded = errors.New("auction has already ended")
)
