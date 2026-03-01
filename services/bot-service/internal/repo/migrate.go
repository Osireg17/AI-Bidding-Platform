package repo

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS bot_bids (
			id         BIGSERIAL PRIMARY KEY,
			bot_id     BIGINT NOT NULL,
			auction_id BIGINT NOT NULL,
			amount     DOUBLE PRECISION NOT NULL,
			status     TEXT NOT NULL DEFAULT 'placed',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_bids_bot_auction ON bot_bids(bot_id, auction_id)`,
	}

	for _, stmt := range ddl {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
