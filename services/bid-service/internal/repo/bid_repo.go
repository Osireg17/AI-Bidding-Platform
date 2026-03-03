package repo

import (
	"context"
	"database/sql"
	"errors"

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

func (r *PostgresBidRepo) GetWinner(ctx context.Context, auctionID int64) (int64, float64, error) {
	var result struct {
		BotID  int64   `bun:"bot_id"`
		Amount float64 `bun:"amount"`
	}
	err := r.db.NewRaw(`
	SELECT bot_id, amount
	FROM bids
	WHERE auction_id = ? AND status = 'accepted'
	ORDER BY amount DESC
	LIMIT 1`, auctionID).Scan(ctx, &result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return result.BotID, result.Amount, nil
}

func (r *PostgresBidRepo) ListByAuction(ctx context.Context, auctionID int64) ([]*domain.Bid, error) {
	var bids []*domain.Bid
	err := r.db.NewSelect().Model(&bids).
		Where("auction_id = ?", auctionID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	return bids, err
}
