package repo

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS auctions (
			id            BIGSERIAL PRIMARY KEY,
			title         TEXT NOT NULL,
			description   TEXT NOT NULL DEFAULT '',
			start_price   DOUBLE PRECISION NOT NULL,
			current_price DOUBLE PRECISION NOT NULL,
			status        TEXT NOT NULL DEFAULT 'pending',
			winner_bot_id BIGINT NOT NULL DEFAULT 0,
			start_time    TIMESTAMPTZ NOT NULL,
			end_time      TIMESTAMPTZ NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auctions_status ON auctions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_auctions_end_time ON auctions(end_time)`,
		`CREATE INDEX IF NOT EXISTS idx_auctions_status_end_time ON auctions(status, end_time)`,
	}

	for _, stmt := range ddl {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
