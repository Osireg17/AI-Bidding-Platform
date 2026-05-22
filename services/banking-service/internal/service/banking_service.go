package service

import (
	"context"
	"database/sql"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// TxRunner abstracts bun.DB.RunInTx so the service is testable without a real DB connection.
type TxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context, tx bun.Tx) error) error
}

type BankingService struct {
	tx         TxRunner
	walletRepo domain.WalletRepository
	itemRepo   domain.ItemRepository
	logger     *zap.Logger
}

func NewBankingService(db *bun.DB, walletRepo domain.WalletRepository, itemRepo domain.ItemRepository, logger *zap.Logger) *BankingService {
	return &BankingService{
		tx:         db,
		walletRepo: walletRepo,
		itemRepo:   itemRepo,
		logger:     logger,
	}
}

func (s *BankingService) Buyout(ctx context.Context, itemID int64) (newBalance float64, err error) {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return 0, err
	}
	if item == nil {
		return 0, domain.ErrItemNotFound
	}

	wallet, err := s.walletRepo.GetByBotID(ctx, item.BotID)
	if err != nil {
		return 0, err
	}

	if wallet == nil {
		return 0, domain.ErrWalletNotFound
	}

	payout := item.PurchasePrice * 0.70
	computed := wallet.Balance + payout

	err = s.tx.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := s.walletRepo.WithTx(tx).UpdateBalance(ctx, item.BotID, computed); err != nil {
			return err
		}
		return s.itemRepo.WithTx(tx).Delete(ctx, itemID)
	})
	if err != nil {
		return 0, err
	}

	return computed, nil
}

func (s *BankingService) RecordWin(ctx context.Context, botID, auctionID int64, title string, winningBid float64) (newBalance float64, err error) {
	wallet, err := s.walletRepo.GetByBotID(ctx, botID)
	if err != nil {
		return 0, err
	}

	if wallet == nil {
		return 0, domain.ErrWalletNotFound
	}

	computed := wallet.Balance - winningBid

	if computed < 0 {
		s.logger.Warn("negative balance after recording win",
			zap.Int64("bot_id", botID),
			zap.Float64("winning_bid", winningBid),
			zap.Float64("new_balance", computed),
		)
	}

	item, err := domain.NewItem(botID, auctionID, title, winningBid)
	if err != nil {
		return 0, err
	}

	err = s.tx.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := s.walletRepo.WithTx(tx).UpdateBalance(ctx, botID, computed); err != nil {
			return err
		}
		return s.itemRepo.WithTx(tx).Create(ctx, item)
	})
	if err != nil {
		return 0, err
	}

	return computed, nil
}

func (s *BankingService) SeedWallets(ctx context.Context) error {
	for botID := int64(1); botID <= 4; botID++ {
		wallet, err := domain.NewWallet(botID, 1_000_000.0)
		if err != nil {
			return err
		}
		err = s.walletRepo.Upsert(ctx, wallet)
		if err != nil {
			s.logger.Error("failed to seed wallet",
				zap.Int64("bot_id", botID),
				zap.Error(err),
			)
			return err
		}
	}
	return nil
}

func (s *BankingService) GetWallet(ctx context.Context, botID int64) (*domain.WalletResponse, error) {
	wallet, err := s.walletRepo.GetByBotID(ctx, botID)
	if err != nil {
		return nil, err
	}

	if wallet == nil {
		return nil, domain.ErrWalletNotFound
	}

	items, err := s.itemRepo.ListByBotID(ctx, botID)
	if err != nil {
		return nil, err
	}

	itemSummaries := make([]*domain.ItemSummary, len(items))
	for i, item := range items {
		itemSummaries[i] = &domain.ItemSummary{
			ItemID:        item.ID,
			Title:         item.Title,
			PurchasePrice: item.PurchasePrice,
			AcquiredAt:    item.AcquiredAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &domain.WalletResponse{
		BotID:   wallet.BotID,
		Balance: wallet.Balance,
		Items:   itemSummaries,
	}, nil
}
