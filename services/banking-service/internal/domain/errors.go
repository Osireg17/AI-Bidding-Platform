package domain

import "errors"

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrItemNotFound         = errors.New("item not found")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrItemNotOwnedByBot    = errors.New("item not owned by bot")
	ErrInvalidWalletData    = errors.New("invalid wallet data")
	ErrInvalidBotID         = errors.New("invalid bot ID")
	ErrInvalidAuctionID     = errors.New("invalid auction ID")
	ErrInvalidTitle         = errors.New("invalid title")
	ErrInvalidPurchasePrice = errors.New("invalid purchase price")
)
