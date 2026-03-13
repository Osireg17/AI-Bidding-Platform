package store

import (
	"strconv"
	"sync"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/shared/events"
	"go.uber.org/zap"
)

type InMemoryStateStore struct {
	mu     sync.RWMutex
	state  domain.AuctionState
	logger *zap.Logger
}

func NewInMemoryStateStore(logger *zap.Logger) *InMemoryStateStore {
	return &InMemoryStateStore{
		logger: logger,
	}
}

func (s *InMemoryStateStore) GetState() domain.AuctionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateCopy := s.state
	stateCopy.Bids = make([]domain.BidView, len(s.state.Bids))
	copy(stateCopy.Bids, s.state.Bids)

	return stateCopy
}

func (s *InMemoryStateStore) ApplyAuctionCreated(payload events.AuctionCreatedPayload) {

	endTime := time.Time{}
	parsedEndTime, err := time.Parse(time.RFC3339, payload.EndTime)
	if err != nil {
		s.logger.Error("failed to parse end time", zap.String("end_time", payload.EndTime), zap.Error(err))
	} else {
		endTime = parsedEndTime
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Auction = domain.AuctionView{
		ID:           payload.AuctionID,
		Title:        payload.Title,
		Description:  payload.Description,
		StartPrice:   payload.StartPrice,
		CurrentPrice: payload.StartPrice,
		Status:       "active",
		EndTime:      endTime,
	}
	s.state.HasAuction = true
	s.state.Bids = nil
	s.state.Winner = domain.WinnerView{}
	s.state.HasWinner = false

}

func (s *InMemoryStateStore) ApplyAuctionEndingSoon(payload events.AuctionEndingSoonPayload) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.state.HasAuction {
		return
	}

	s.state.Auction.Status = "ending_soon"

}

func (s *InMemoryStateStore) ApplyAuctionEnded(payload events.AuctionEndedPayload) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.state.HasAuction {
		return
	}

	s.state.Auction.Status = "closed"

	botName := domain.BotNames[payload.WinnerBotID]

	s.state.Winner = domain.WinnerView{
		BotName:     botName,
		BotID:       payload.WinnerBotID,
		Amount:      payload.WinningBid,
		FinalStatus: payload.FinalStatus,
	}
	s.state.HasWinner = true

}

func (s *InMemoryStateStore) ApplyBidPlaced(payload events.BidPlacedPayload) {

	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		s.logger.Error("failed to parse bid timestamp", zap.String("timestamp", payload.Timestamp), zap.Error(err))
		timestamp = time.Time{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.state.HasAuction {
		return
	}

	botName := domain.BotNames[payload.BotID]
	if botName == "" {
		botName = "Bot #" + strconv.FormatInt(payload.BotID, 10)
	}

	bid := domain.BidView{
		BotName:   botName,
		BotID:     payload.BotID,
		Amount:    payload.BidAmount,
		Timestamp: timestamp,
	}

	s.state.Bids = append([]domain.BidView{bid}, s.state.Bids...)
	s.state.Auction.CurrentPrice = payload.BidAmount

}

var _ domain.StateStore = (*InMemoryStateStore)(nil)
