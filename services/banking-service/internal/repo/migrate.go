package repo

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*domain.Wallet)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create wallets table: %w", err)
	}

	_, err = db.NewCreateTable().Model((*domain.Item)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create items table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_items_bot_id ON items(bot_id)",
		"CREATE INDEX IF NOT EXISTS idx_items_auction_id ON items(auction_id)",
	}
	for _, ddl := range indexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
