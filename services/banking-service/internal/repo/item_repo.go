package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/uptrace/bun"
)

type PostgresItemRepo struct {
	db bun.IDB
}

func NewPostgresItemRepo(db *bun.DB) *PostgresItemRepo {
	return &PostgresItemRepo{db: db}
}

func (r *PostgresItemRepo) WithTx(tx bun.IDB) domain.ItemRepository {
	return &PostgresItemRepo{db: tx}
}

func (r *PostgresItemRepo) Create(ctx context.Context, item *domain.Item) error {
	_, err := r.db.NewInsert().Model(item).Exec(ctx)
	return err
}

func (r *PostgresItemRepo) GetByID(ctx context.Context, itemID int64) (*domain.Item, error) {
	var item domain.Item
	err := r.db.NewSelect().Model(&item).Where("id = ?", itemID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *PostgresItemRepo) ListByBotID(ctx context.Context, botID int64) ([]*domain.Item, error) {
	var items []*domain.Item
	err := r.db.NewSelect().Model(&items).Where("bot_id = ?", botID).OrderExpr("acquired_at DESC").Scan(ctx)
	return items, err
}

func (r *PostgresItemRepo) Delete(ctx context.Context, itemID int64) error {
	_, err := r.db.NewDelete().Model((*domain.Item)(nil)).Where("id = ?", itemID).Exec(ctx)
	return err
}
