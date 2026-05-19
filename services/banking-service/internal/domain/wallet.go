package domain

import (
	"time"

	"github.com/uptrace/bun"
)

type Wallet struct {
	bun.BaseModel `bun:"table:wallets,alias:w"`
	BotID         int64     `bun:",pk,notnull"`
	Balance       float64   `bun:",notnull"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

func NewWallet(botID int64, initialBalance float64) (*Wallet, error) {
	if botID == 0 {
		return nil, ErrInvalidWalletData
	}
	if initialBalance < 0 {
		return nil, ErrInvalidWalletData
	}

	return &Wallet{
		BotID:     botID,
		Balance:   initialBalance,
		UpdatedAt: time.Now().UTC(),
	}, nil
}
