package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresAuctionRepo struct {
	db *bun.DB
}

func NewPostgresAuctionRepo(db *bun.DB) *PostgresAuctionRepo {
	return &PostgresAuctionRepo{db: db}
}

func (r *PostgresAuctionRepo) Create(ctx context.Context, auction *domain.Auction) error {
	_, err := r.db.NewInsert().Model(auction).Returning("id").Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auction %d: %w", auction.ID, err)
	}
	return nil
}

func (r *PostgresAuctionRepo) GetByID(ctx context.Context, id int64) (*domain.Auction, error) {
	auction := new(domain.Auction)
	err := r.db.NewSelect().Model(auction).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, domain.ErrAuctionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get auction %d: %w", id, err)
	}
	return auction, nil
}

func (r *PostgresAuctionRepo) List(ctx context.Context) ([]*domain.Auction, error) {
	var auctions []*domain.Auction
	err := r.db.NewSelect().Model(&auctions).Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list auctions: %w", err)
	}
	return auctions, nil
}

func (r *PostgresAuctionRepo) Update(ctx context.Context, auction *domain.Auction) error {
	result, err := r.db.NewUpdate().Model(auction).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update auction %d: %w", auction.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for auction %d: %w", auction.ID, err)
	}
	if rowsAffected == 0 {
		return domain.ErrAuctionNotFound
	}
	return nil
}

func (r *PostgresAuctionRepo) FindExpiredActive(ctx context.Context) ([]*domain.Auction, error) {
	var auctions []*domain.Auction
	err := r.db.NewSelect().
		Model(&auctions).
		Where("status IN (?)", bun.In([]string{"active", "ending_soon"})).
		Where("end_time < NOW()").
		Order("end_time ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired active auctions: %w", err)
	}
	return auctions, nil
}

func (r *PostgresAuctionRepo) FindEndingSoon(ctx context.Context, thresholdSeconds int) ([]*domain.Auction, error) {
	var auctions []*domain.Auction
	err := r.db.NewSelect().
		Model(&auctions).
		Where("status = ?", "active").
		Where("end_time > NOW()").
		Where("end_time <= NOW() + (? || ' seconds')::interval", fmt.Sprintf("%d", thresholdSeconds)).
		Order("end_time ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find ending-soon auctions: %w", err)
	}
	return auctions, nil
}
