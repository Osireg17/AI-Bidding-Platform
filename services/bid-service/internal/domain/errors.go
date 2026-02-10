package domain

import "errors"

var (
	ErrBidTooLow             = errors.New("bid amount is too low")
	ErrAuctionNotActive      = errors.New("auction is not active")
	ErrAuctionNotFound       = errors.New("auction not found")
	ErrInvalidBidData        = errors.New("invalid bid data")
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock for auction")
)
