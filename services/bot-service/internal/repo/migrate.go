package repo

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/domain"
	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*domain.BotBid)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create bot_bids table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_bot_bids_bot_auction ON bot_bids(bot_id, auction_id)",
	}
	for _, ddl := range indexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
