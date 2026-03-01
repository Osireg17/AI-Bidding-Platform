package domain

import "errors"

var (
	ErrAlreadyBid        = errors.New("bot has already placed a bid on this auction")
	ErrBotNotFound       = errors.New("bot not found")
	ErrInvalidBotData    = errors.New("invalid bot data")
	ErrInvalidBotBidData = errors.New("invalid bot bid data")
)
