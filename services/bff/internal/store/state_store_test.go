package store

import (
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		logger: zap.NewNop(),
	}
}

func TestApplyAuctionCreated(t *testing.T) {
	s := newTestStore()

	// Seed old state to prove ApplyAuctionCreated resets it.
	s.state.HasAuction = true
	s.state.Bids = []domain.BidView{{BotName: "Bot1", BotID: 1, Amount: 100}}
	s.state.HasWinner = true
	s.state.Winner = domain.WinnerView{BotName: "Bot1", BotID: 1, Amount: 100}

	end := "2024-06-30T15:04:05Z"
	payload := events.AuctionCreatedPayload{
		AuctionID:   123,
		Title:       "Test Auction",
		Description: "A test auction",
		StartPrice:  50.0,
		EndTime:     end,
	}

	s.ApplyAuctionCreated(payload)

	wantEnd, err := time.Parse(time.RFC3339, end)
	require.NoError(t, err)

	want := domain.AuctionState{
		Auction: domain.AuctionView{
			ID:           123,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   50.0,
			CurrentPrice: 50.0,
			Status:       "active",
			EndTime:      wantEnd,
		},
		HasAuction: true,
		Bids:       []domain.BidView{},
		Winner:     domain.WinnerView{},
		HasWinner:  false,
	}

	got := s.GetState()
	assert.Equal(t, want, got)
}

func TestApplyAuctionEndingSoon(t *testing.T) {
	s := newTestStore()
	s.state.HasAuction = true

	payload := events.AuctionEndingSoonPayload{
		AuctionID: 123,
		EndTime:   "2024-06-30T15:04:05Z",
	}
	s.ApplyAuctionEndingSoon(payload)

	wantStatus := "ending_soon"
	got := s.GetState()
	assert.Equal(t, wantStatus, got.Auction.Status)
}

func TestApplyAuctionEnded(t *testing.T) {
	s := newTestStore()
	wantEnd, err := time.Parse(time.RFC3339, "2024-06-30T15:04:05Z")
	require.NoError(t, err)

	s.state = domain.AuctionState{
		Auction: domain.AuctionView{
			ID:           123,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   50,
			CurrentPrice: 100,
			Status:       "active",
			EndTime:      wantEnd,
		},
		HasAuction: true,
		Bids:       []domain.BidView{{BotName: "Bot1", BotID: 1, Amount: 100}},
	}

	payload := events.AuctionEndedPayload{
		AuctionID:   123,
		WinnerBotID: 1,
		WinningBid:  100,
		TotalBids:   1,
		FinalStatus: "sold",
	}

	s.ApplyAuctionEnded(payload)

	want := domain.AuctionState{
		Auction: domain.AuctionView{
			ID:           123,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   50,
			CurrentPrice: 100,
			Status:       "closed",
			EndTime:      wantEnd,
		},
		HasAuction: true,
		Bids:       []domain.BidView{{BotName: "Bot1", BotID: 1, Amount: 100}},
		Winner: domain.WinnerView{
			BotName:     "Aggressive Alice",
			BotID:       1,
			Amount:      100,
			FinalStatus: "sold",
		},
		HasWinner: true,
	}

	got := s.GetState()
	assert.Equal(t, want, got)
}

func TestApplyBidPlaced(t *testing.T) {
	s := newTestStore()
	wantEnd, err := time.Parse(time.RFC3339, "2024-06-30T15:04:05Z")
	require.NoError(t, err)

	s.state = domain.AuctionState{
		Auction: domain.AuctionView{
			ID:           123,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   50,
			CurrentPrice: 50,
			Status:       "active",
			EndTime:      wantEnd,
		},
		HasAuction: true,
	}

	payload := events.BidPlacedPayload{
		AuctionID: 123,
		BotID:     2,
		BidAmount: 60,
		Timestamp: "2024-06-30T12:00:00Z",
	}

	s.ApplyBidPlaced(payload)

	want := domain.AuctionState{
		Auction: domain.AuctionView{
			ID:           123,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   50,
			CurrentPrice: 60,
			Status:       "active",
			EndTime:      wantEnd,
		},
		HasAuction: true,
		Bids: []domain.BidView{
			{
				BotName:   "Sniper Steve",
				BotID:     2,
				Amount:    60,
				Timestamp: time.Date(2024, 6, 30, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	got := s.GetState()
	assert.Equal(t, want, got)
}
