package domain

import "context"

type WalletRespository interface {
	Upsert(ctx context.Context, wallet *Wallet) error

	GetByBotID(ctx context.Context, botID int64) (*Wallet, error)

	UpdateBalance(ctx context.Context, botID int64, amount float64) error
}

type ItemRepository interface {
	Create(ctx context.Context, item *Item) error

	GetByID(ctx context.Context, itemID int64) (*Item, error)

	ListByBotID(ctx context.Context, botID int64) ([]*Item, error)

	Delete(ctx context.Context, itemID int64) error
}

type BankingService interface {
	GetWallet(ctx context.Context, botID int64) (*Wallet, error)

	Buyout(ctx context.Context, itemID int64) (newBalance float64, err error)

	RecordWin(ctx context.Context, auctionID, botID int64, winningBid float64, title string) error

	SeedWallet(ctx context.Context) error
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
