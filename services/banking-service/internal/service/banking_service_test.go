package service

import (
	"context"
	"testing"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func NewTestService(
	walletRepo *testutil.MockWalletRepository,
	itemRepo *testutil.MockItemRepository,
	t *testing.T,
) *BankingService {
	t.Helper()
	logger := zap.NewNop()
	svc := NewBankingService(nil, walletRepo, itemRepo, logger)
	svc.tx = &testutil.MockTxRunner{}
	return svc
}

func TestBankingService_Buyout(t *testing.T) {
	ctx := context.Background()
	wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(100), testutil.WithWalletBalance(1000))
	item := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemPurchasePrice(500))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = wallet
	itemRepo.GetByIDResult = item

	svc := NewTestService(walletRepo, itemRepo, t)

	newBalance, err := svc.Buyout(ctx, item.ID)
	require.NoError(t, err)

	expectedPayout := item.PurchasePrice * 0.70
	expectedNewBalance := wallet.Balance + expectedPayout
	require.Equal(t, expectedNewBalance, newBalance)

	require.Len(t, walletRepo.UpdateBalanceArgs, 1)
	require.Equal(t, item.BotID, walletRepo.UpdateBalanceArgs[0].BotID)
	require.Equal(t, expectedNewBalance, walletRepo.UpdateBalanceArgs[0].NewBalance)

	require.Len(t, itemRepo.DeleteArgs, 1)
	require.Equal(t, item.ID, itemRepo.DeleteArgs[0].ItemID)
}

func TestBankingService_Buyout_ItemNotFound(t *testing.T) {
	ctx := context.Background()

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	itemRepo.GetByIDResult = nil

	svc := NewTestService(walletRepo, itemRepo, t)

	_, err := svc.Buyout(ctx, 999)
	require.ErrorIs(t, err, domain.ErrItemNotFound)
}

func TestBankingService_Buyout_WalletNotFound(t *testing.T) {
	ctx := context.Background()
	item := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemPurchasePrice(500))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	itemRepo.GetByIDResult = item
	walletRepo.GetByBotIDResult = nil

	svc := NewTestService(walletRepo, itemRepo, t)

	_, err := svc.Buyout(ctx, item.ID)
	require.ErrorIs(t, err, domain.ErrWalletNotFound)
}

func TestBankingService_Buyout_UpdateBalanceError(t *testing.T) {
	ctx := context.Background()
	wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(100), testutil.WithWalletBalance(1000))
	item := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemPurchasePrice(500))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = wallet
	walletRepo.UpdateBalanceErr = assert.AnError
	itemRepo.GetByIDResult = item

	svc := NewTestService(walletRepo, itemRepo, t)

	_, err := svc.Buyout(ctx, item.ID)
	require.ErrorIs(t, err, assert.AnError)
}

func TestBankingService_RecordWin(t *testing.T) {
	ctx := context.Background()
	wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(100), testutil.WithWalletBalance(1000))
	item := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemPurchasePrice(500))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = wallet

	svc := NewTestService(walletRepo, itemRepo, t)

	winningBid := 300.0
	newBalance, err := svc.RecordWin(ctx, wallet.BotID, item.AuctionID, item.Title, winningBid)
	require.NoError(t, err)

	expectedNewBalance := wallet.Balance - winningBid
	assert.Equal(t, expectedNewBalance, newBalance)

	require.Len(t, walletRepo.UpdateBalanceArgs, 1)
	require.Equal(t, wallet.BotID, walletRepo.UpdateBalanceArgs[0].BotID)
	require.Equal(t, expectedNewBalance, walletRepo.UpdateBalanceArgs[0].NewBalance)
}

func TestBankingService_RecordWin_NegativeBalance(t *testing.T) {
	ctx := context.Background()
	wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(100), testutil.WithWalletBalance(100))
	item := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemPurchasePrice(500))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = wallet

	svc := NewTestService(walletRepo, itemRepo, t)

	winningBid := 150.0
	newBalance, err := svc.RecordWin(ctx, wallet.BotID, item.AuctionID, item.Title, winningBid)
	require.NoError(t, err)

	expectedNewBalance := wallet.Balance - winningBid
	assert.Equal(t, expectedNewBalance, newBalance)

	require.Len(t, walletRepo.UpdateBalanceArgs, 1)
	require.Equal(t, wallet.BotID, walletRepo.UpdateBalanceArgs[0].BotID)
	require.Equal(t, expectedNewBalance, walletRepo.UpdateBalanceArgs[0].NewBalance)
}

func TestBankingService_SeedWallets(t *testing.T) {
	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}

	svc := NewTestService(walletRepo, itemRepo, t)

	err := svc.SeedWallets(context.Background())
	require.NoError(t, err)

	require.Equal(t, 4, walletRepo.UpsertCalls)
	for i := 1; i <= 4; i++ {
		found := false
		for _, arg := range walletRepo.UpsertArgs {
			if arg.Wallet.BotID == int64(i) && arg.Wallet.Balance == 1_000_000.0 {
				found = true
				break
			}
		}
		assert.True(t, found, "expected wallet for bot ID %d not found", i)
	}
}

func TestBankingService_GetWallet(t *testing.T) {
	ctx := context.Background()
	wallet := testutil.CreateTestWallet(testutil.WithWalletBotID(100), testutil.WithWalletBalance(1000))
	item1 := testutil.CreateTestItem(testutil.WithItemID(1), testutil.WithItemBotID(100), testutil.WithItemTitle("Item 1"), testutil.WithItemPurchasePrice(200))
	item2 := testutil.CreateTestItem(testutil.WithItemID(2), testutil.WithItemBotID(100), testutil.WithItemTitle("Item 2"), testutil.WithItemPurchasePrice(300))

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = wallet
	itemRepo.ListByBotIDResult = []*domain.Item{item1, item2}

	svc := NewTestService(walletRepo, itemRepo, t)

	resp, err := svc.GetWallet(ctx, wallet.BotID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, wallet.BotID, resp.BotID)
	assert.Equal(t, wallet.Balance, resp.Balance)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, item1.ID, resp.Items[0].ItemID)
	assert.Equal(t, item1.Title, resp.Items[0].Title)
	assert.Equal(t, item1.PurchasePrice, resp.Items[0].PurchasePrice)
	assert.Equal(t, item1.AcquiredAt.Format("2006-01-02 15:04:05"), resp.Items[0].AcquiredAt)
	assert.Equal(t, item2.ID, resp.Items[1].ItemID)
	assert.Equal(t, item2.Title, resp.Items[1].Title)
	assert.Equal(t, item2.PurchasePrice, resp.Items[1].PurchasePrice)
	assert.Equal(t, item2.AcquiredAt.Format("2006-01-02 15:04:05"), resp.Items[1].AcquiredAt)
}

func TestBankingService_GetWallet_WalletNotFound(t *testing.T) {
	ctx := context.Background()

	walletRepo := &testutil.MockWalletRepository{}
	itemRepo := &testutil.MockItemRepository{}
	walletRepo.GetByBotIDResult = nil

	svc := NewTestService(walletRepo, itemRepo, t)

	_, err := svc.GetWallet(ctx, 999)
	require.ErrorIs(t, err, domain.ErrWalletNotFound)
}
