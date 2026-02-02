package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/uptrace/bun"
)

// === CONTEXT ===
// Purpose: PostgreSQL implementation of domain.AuctionRepository using Bun ORM.
// This is the adapter side of Ports & Adapters — it implements the port defined in domain/ports.go.
//
// === DEPENDENCIES ===
// bun — SQL-first ORM, wraps database/sql with type-safe query builder
// domain — for Auction model (with bun struct tags) and AuctionRepository interface
//
// === DATA / STATE ===
// PostgresAuctionRepo holds a *bun.DB connection. Created once at startup, closed at shutdown.

// PostgresAuctionRepo implements domain.AuctionRepository using Bun + PostgreSQL.
type PostgresAuctionRepo struct {
	db *bun.DB
}

// NewPostgresAuctionRepo creates a new repository with the given Bun database connection.
func NewPostgresAuctionRepo(db *bun.DB) *PostgresAuctionRepo {
	return &PostgresAuctionRepo{db: db}
}

// === BEHAVIOR: Create ===
// Input: context, *Auction
// Output: error if insert fails
// Logic: INSERT the auction model into the auctions table

func (r *PostgresAuctionRepo) Create(ctx context.Context, auction *domain.Auction) error {
	_, err := r.db.NewInsert().Model(auction).Returning("id").Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auction %d: %w", auction.ID, err)
	}
	return nil
}

// === BEHAVIOR: GetByID ===
// Input: context, auction ID string
// Output: *Auction or ErrAuctionNotFound if no row matches

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

// === BEHAVIOR: List ===
// Input: context
// Output: slice of all auctions ordered by created_at DESC

func (r *PostgresAuctionRepo) List(ctx context.Context) ([]*domain.Auction, error) {
	var auctions []*domain.Auction
	err := r.db.NewSelect().Model(&auctions).Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list auctions: %w", err)
	}
	return auctions, nil
}

// === BEHAVIOR: Update ===
// Input: context, *Auction with modified fields
// Output: error if update fails or auction not found

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

// === BEHAVIOR: FindExpiredActive ===
// Input: context
// Output: auctions with status active/ending_soon whose end_time is in the past
// Note: Returns at most 1 in practice (single-auction-at-a-time design), but returns a slice for correctness.

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

// === BEHAVIOR: FindEndingSoon ===
// Input: context, thresholdSeconds (how many seconds before end_time counts as "ending soon")
// Output: active auctions within the ending-soon window that haven't been marked yet

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
