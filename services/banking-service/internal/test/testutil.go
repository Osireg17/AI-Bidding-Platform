package test

import (
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func NewTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zaptest.NewLogger(t)
}

type WalletOption func(*domain.Wallet)
type ItemOption func(*domain.Item)

func CreateTestWallet(opts ...WalletOption) *domain.Wallet {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	wallet := &domain.Wallet{
		BotID:     100,
		Balance:   1000.0,
		UpdatedAt: now,
	}
	for _, opt := range opts {
		opt(wallet)
	}
	return wallet
}

func WithWalletBotID(id int64) WalletOption {
	return func(w *domain.Wallet) { w.BotID = id }
}

func WithWalletBalance(balance float64) WalletOption {
	return func(w *domain.Wallet) { w.Balance = balance }
}

func WithWalletUpdatedAt(t time.Time) WalletOption {
	return func(w *domain.Wallet) { w.UpdatedAt = t }
}

func CreateTestItem(opts ...ItemOption) *domain.Item {
	item := &domain.Item{
		ID:        1,
		AuctionID: 10,
		Title:     "Test Item",
	}
	for _, opt := range opts {
		opt(item)
	}
	return item
}

func WithItemID(id int64) ItemOption {
	return func(i *domain.Item) { i.ID = id }
}

func WithItemAuctionID(id int64) ItemOption {
	return func(i *domain.Item) { i.AuctionID = id }
}

func WithItemTitle(title string) ItemOption {
	return func(i *domain.Item) { i.Title = title }
}

func WithItemPurchasePrice(price float64) ItemOption {
	return func(i *domain.Item) { i.PurchasePrice = price }
}

func WithItemAcquiredAt(t time.Time) ItemOption {
	return func(i *domain.Item) { i.AcquiredAt = t }
}

func WithItemBotID(id int64) ItemOption {
	return func(i *domain.Item) { i.BotID = id }
}
