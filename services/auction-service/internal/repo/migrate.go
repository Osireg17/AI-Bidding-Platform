package repo

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*domain.Auction)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auctions table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_auctions_status ON auctions(status)",
		"CREATE INDEX IF NOT EXISTS idx_auctions_end_time ON auctions(end_time)",
		"CREATE INDEX IF NOT EXISTS idx_auctions_status_end_time ON auctions(status, end_time)",
	}
	for _, ddl := range indexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
