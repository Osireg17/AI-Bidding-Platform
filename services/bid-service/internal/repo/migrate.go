package repo

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/uptrace/bun"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*domain.Bid)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	_, err = db.NewCreateTable().Model((*domain.AuctionSnapshot)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_bids_auction_id ON bids(auction_id)",
		"CREATE INDEX IF NOT EXISTS idx_bids_auction_amount ON bids(auction_id, status, amount DESC)",
	}
	for _, ddl := range indexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return err
		}
	}

	return nil
}
