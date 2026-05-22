package domain

import (
	"context"

	"github.com/uptrace/bun"
)

type WalletRepository interface {
	Upsert(ctx context.Context, wallet *Wallet) error
	GetByBotID(ctx context.Context, botID int64) (*Wallet, error)
	UpdateBalance(ctx context.Context, botID int64, newBalance float64) error
	WithTx(tx bun.IDB) WalletRepository
}

type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	GetByID(ctx context.Context, itemID int64) (*Item, error)
	ListByBotID(ctx context.Context, botID int64) ([]*Item, error)
	Delete(ctx context.Context, itemID int64) error
	WithTx(tx bun.IDB) ItemRepository
}

type BankingService interface {
	GetWallet(ctx context.Context, botID int64) (*WalletResponse, error)
	Buyout(ctx context.Context, itemID int64) (newBalance float64, err error)
	RecordWin(ctx context.Context, botID, auctionID int64, title string, winningBid float64) (newBalance float64, err error)
	SeedWallets(ctx context.Context) error
}

type WalletResponse struct {
	BotID   int64
	Balance float64
	Items   []*ItemSummary
}

type ItemSummary struct {
	ItemID        int64
	Title         string
	PurchasePrice float64
	AcquiredAt    string
}
