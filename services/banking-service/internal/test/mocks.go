package test

import (
	"context"
	"database/sql"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/uptrace/bun"
)

type MockWalletRepository struct {
	UpsertCalls int
	UpsertArgs  []UpsertArgs
	UpsertErr   error

	GetByBotIDCalls  int
	GetByBotIDArgs   []GetByBotIDArgs
	GetByBotIDResult *domain.Wallet
	GetByBotIDErr    error

	UpdateBalanceCalls int
	UpdateBalanceArgs  []UpdateBalanceArgs
	UpdateBalanceErr   error
}

type MockItemRepository struct {
	CreateCalls int
	CreateArgs  []CreateItemArgs
	CreateErr   error

	GetByIDCalls  int
	GetByIDArgs   []GetByIDArgs
	GetByIDResult *domain.Item
	GetByIDErr    error

	ListByBotIDCalls  int
	ListByBotIDArgs   []ListByBotIDArgs
	ListByBotIDResult []*domain.Item
	ListByBotIDErr    error

	DeleteCalls int
	DeleteArgs  []DeleteArgs
	DeleteErr   error
}

type UpsertArgs struct {
	Ctx    context.Context
	Wallet *domain.Wallet
}

type GetByBotIDArgs struct {
	Ctx   context.Context
	BotID int64
}

type UpdateBalanceArgs struct {
	Ctx        context.Context
	BotID      int64
	NewBalance float64
}

type CreateItemArgs struct {
	Ctx  context.Context
	Item *domain.Item
}

func (m *MockWalletRepository) Upsert(ctx context.Context, wallet *domain.Wallet) error {
	m.UpsertCalls++
	m.UpsertArgs = append(m.UpsertArgs, UpsertArgs{Ctx: ctx, Wallet: wallet})
	return m.UpsertErr
}

func (m *MockWalletRepository) GetByBotID(ctx context.Context, botID int64) (*domain.Wallet, error) {
	m.GetByBotIDCalls++
	m.GetByBotIDArgs = append(m.GetByBotIDArgs, GetByBotIDArgs{Ctx: ctx, BotID: botID})
	return m.GetByBotIDResult, m.GetByBotIDErr
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, botID int64, newBalance float64) error {
	m.UpdateBalanceCalls++
	m.UpdateBalanceArgs = append(m.UpdateBalanceArgs, UpdateBalanceArgs{Ctx: ctx, BotID: botID, NewBalance: newBalance})
	return m.UpdateBalanceErr
}

func (m *MockWalletRepository) WithTx(_ bun.IDB) domain.WalletRepository { return m }

func (m *MockItemRepository) Create(ctx context.Context, item *domain.Item) error {
	m.CreateCalls++
	m.CreateArgs = append(m.CreateArgs, CreateItemArgs{Ctx: ctx, Item: item})
	return m.CreateErr
}

type GetByIDArgs struct {
	Ctx    context.Context
	ItemID int64
}

type DeleteArgs struct {
	Ctx    context.Context
	ItemID int64
}

func (m *MockItemRepository) GetByID(ctx context.Context, itemID int64) (*domain.Item, error) {
	m.GetByIDCalls++
	m.GetByIDArgs = append(m.GetByIDArgs, GetByIDArgs{Ctx: ctx, ItemID: itemID})
	return m.GetByIDResult, m.GetByIDErr
}

type ListByBotIDArgs struct {
	Ctx   context.Context
	BotID int64
}

func (m *MockItemRepository) ListByBotID(ctx context.Context, botID int64) ([]*domain.Item, error) {
	m.ListByBotIDCalls++
	m.ListByBotIDArgs = append(m.ListByBotIDArgs, ListByBotIDArgs{Ctx: ctx, BotID: botID})
	return m.ListByBotIDResult, m.ListByBotIDErr
}

func (m *MockItemRepository) Delete(ctx context.Context, itemID int64) error {
	m.DeleteCalls++
	m.DeleteArgs = append(m.DeleteArgs, DeleteArgs{Ctx: ctx, ItemID: itemID})
	return m.DeleteErr
}

func (m *MockItemRepository) WithTx(_ bun.IDB) domain.ItemRepository { return m }

// MockTxRunner executes the closure directly without a real DB transaction.
// Used in unit tests so the service's RunInTx calls work without a Postgres connection.
type MockTxRunner struct{}

func (m *MockTxRunner) RunInTx(ctx context.Context, _ *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	return fn(ctx, bun.Tx{})
}

// Compile-time interface guards.
var _ domain.WalletRepository = (*MockWalletRepository)(nil)
var _ domain.ItemRepository = (*MockItemRepository)(nil)
