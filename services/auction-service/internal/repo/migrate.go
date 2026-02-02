package repo

import (
	"context"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/uptrace/bun"
)

// === CONTEXT ===
// Purpose: Run database migrations at application startup using Bun's model-based table creation.
// The Auction struct's bun tags define the schema. Indexes are created via raw SQL since
// Bun's CreateTable doesn't support index creation directly.
//
// === BEHAVIOR: RunMigrations ===
// Input: context, *bun.DB
// Output: error if migration fails
// Preconditions: DB connection is alive
// Postconditions: auctions table and indexes exist (idempotent via IfNotExists)
// Logic:
//   CREATE table from Auction model (IF NOT EXISTS)
//   CREATE indexes for scheduler queries (IF NOT EXISTS)

// RunMigrations creates the auctions table and indexes. Safe to call on every startup.
func RunMigrations(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().
		Model((*domain.Auction)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auctions table: %w", err)
	}

	// Create indexes for the scheduler queries.
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_auctions_status ON auctions (status)",
		"CREATE INDEX IF NOT EXISTS idx_auctions_end_time ON auctions (end_time)",
		"CREATE INDEX IF NOT EXISTS idx_auctions_status_end_time ON auctions (status, end_time)",
	}
	for _, ddl := range indexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
