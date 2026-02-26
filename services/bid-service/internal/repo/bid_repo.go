package repo

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresBidRepo struct {
	db *bun.DB
}

func NewPostgresBidRepo(db *bun.DB) *PostgresBidRepo {
	return &PostgresBidRepo{db: db}
}

func (r *PostgresBidRepo) Create(ctx context.Context, bid *domain.Bid) error {
	_, err := r.db.NewInsert().Model(bid).Exec(ctx)
	return err
}

func (r *PostgresBidRepo) GetHighestBid(ctx context.Context, auctionID int64) (float64, error) {
	var highestBid float64
	err := r.db.NewRaw(`
		SELECT COALESCE(MAX(amount), 0)
		FROM bids
		WHERE auction_id = ? AND status = 'accepted'`, auctionID).Scan(ctx, &highestBid)
	return highestBid, err
}

func (r *PostgresBidRepo) ListByAuction(ctx context.Context, auctionID int64) ([]*domain.Bid, error) {
	var bids []*domain.Bid
	err := r.db.NewSelect().Model(&bids).
		Where("auction_id = ?", auctionID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	return bids, err
}
