package repo

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresWalletRepo struct {
	db bun.IDB
}

func NewPostgresWalletRepo(db *bun.DB) *PostgresWalletRepo {
	return &PostgresWalletRepo{db: db}
}

func (r *PostgresWalletRepo) WithTx(tx bun.IDB) domain.WalletRepository {
	return &PostgresWalletRepo{db: tx}
}

func (r *PostgresWalletRepo) Upsert(ctx context.Context, wallet *domain.Wallet) error {
	_, err := r.db.NewInsert().Model(wallet).On("CONFLICT (bot_id) DO UPDATE").Exec(ctx)
	return err
}

func (r *PostgresWalletRepo) GetByBotID(ctx context.Context, botID int64) (*domain.Wallet, error) {
	var wallet domain.Wallet
	err := r.db.NewSelect().Model(&wallet).Where("bot_id = ?", botID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *PostgresWalletRepo) UpdateBalance(ctx context.Context, botID int64, newBalance float64) error {
	_, err := r.db.NewUpdate().Model(&domain.Wallet{}).Where("bot_id = ?", botID).Set("balance = ?", newBalance).Exec(ctx)
	return err
}
