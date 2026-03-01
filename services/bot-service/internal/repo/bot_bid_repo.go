package repo

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresBotBidRepo struct {
	db *bun.DB
}

func NewPostgresBotBidRepo(db *bun.DB) *PostgresBotBidRepo {
	return &PostgresBotBidRepo{db: db}
}

func (r *PostgresBotBidRepo) Create(ctx context.Context, botBid *domain.BotBid) error {
	_, err := r.db.NewInsert().Model(botBid).Exec(ctx)
	return err
}

func (r *PostgresBotBidRepo) GetByAuction(ctx context.Context, auctionID int64) ([]*domain.BotBid, error) {
	var bids []*domain.BotBid
	err := r.db.NewSelect().Model(&bids).
		Where("auction_id = ?", auctionID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	return bids, err
}

func (r *PostgresBotBidRepo) HasBidOnAuction(ctx context.Context, botID, auctionID int64) (bool, error) {
	exists, err := r.db.NewSelect().Model((*domain.BotBid)(nil)).
		Where("bot_id = ? AND auction_id = ?", botID, auctionID).
		Exists(ctx)
	return exists, err
}
