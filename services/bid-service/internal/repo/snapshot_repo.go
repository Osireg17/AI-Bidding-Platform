package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresSnapshotRepo struct {
	db *bun.DB
}

func NewPostgresSnapshotRepo(db *bun.DB) *PostgresSnapshotRepo {
	return &PostgresSnapshotRepo{db: db}
}

func (r *PostgresSnapshotRepo) Upsert(ctx context.Context, snapshot *domain.AuctionSnapshot) error {
	_, err := r.db.NewInsert().Model(snapshot).
		On("CONFLICT (auction_id) DO UPDATE").
		Set("status = EXCLUDED.status, updated_at = NOW()").
		Exec(ctx)
	return err
}

func (r *PostgresSnapshotRepo) GetByID(ctx context.Context, auctionID int64) (*domain.AuctionSnapshot, error) {
	var snapshot domain.AuctionSnapshot
	err := r.db.NewSelect().Model(&snapshot).
		Where("auction_id = ?", auctionID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAuctionNotFound
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *PostgresSnapshotRepo) UpdateStatus(ctx context.Context, auctionID int64, status string) error {
	_, err := r.db.NewUpdate().Model((*domain.AuctionSnapshot)(nil)).
		Set("status = ?", status).
		Where("auction_id = ?", auctionID).
		Exec(ctx)
	return err
}
