package repo

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS bids (
			id         BIGSERIAL PRIMARY KEY,
			auction_id BIGINT NOT NULL,
			bot_id     BIGINT NOT NULL,
			amount     DOUBLE PRECISION NOT NULL,
			status     TEXT NOT NULL,
			reason     TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bids_auction_id ON bids(auction_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bids_auction_amount ON bids(auction_id, status, amount DESC)`,
		`CREATE TABLE IF NOT EXISTS auction_snapshots (
			auction_id BIGINT PRIMARY KEY,
			title      TEXT NOT NULL,
			start_price DOUBLE PRECISION NOT NULL,
			status     TEXT NOT NULL,
			start_time TIMESTAMPTZ NOT NULL,
			end_time   TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, stmt := range ddl {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
