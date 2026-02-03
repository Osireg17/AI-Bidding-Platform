package testutil

import (
	"context"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
)

type MockAuctionRepository struct {
	CreateCalls int
	CreateArgs  []CreateArgs
	CreateErr   error

	GetByIDCalls  int
	GetByIDArgs   []GetByIDArgs
	GetByIDResult *domain.Auction
	GetByIDErr    error

	ListCalls  int
	ListArgs   []ListArgs
	ListResult []*domain.Auction
	ListErr    error

	UpdateCalls int
	UpdateArgs  []UpdateArgs
	UpdateErr   error

	FindExpiredActiveCalls  int
	FindExpiredActiveArgs   []FindExpiredActiveArgs
	FindExpiredActiveResult []*domain.Auction
	FindExpiredActiveErr    error

	FindEndingSoonCalls  int
	FindEndingSoonArgs   []FindEndingSoonArgs
	FindEndingSoonResult []*domain.Auction
	FindEndingSoonErr    error
}

type CreateArgs struct {
	Ctx     context.Context
	Auction *domain.Auction
}

func (m *MockAuctionRepository) Create(ctx context.Context, auction *domain.Auction) error {
	m.CreateCalls++
	m.CreateArgs = append(m.CreateArgs, CreateArgs{Ctx: ctx, Auction: auction})
	return m.CreateErr
}

type GetByIDArgs struct {
	Ctx context.Context
	ID  int64
}

func (m *MockAuctionRepository) GetByID(ctx context.Context, id int64) (*domain.Auction, error) {
	m.GetByIDCalls++
	m.GetByIDArgs = append(m.GetByIDArgs, GetByIDArgs{Ctx: ctx, ID: id})
	return m.GetByIDResult, m.GetByIDErr
}

type ListArgs struct {
	Ctx context.Context
}

func (m *MockAuctionRepository) List(ctx context.Context) ([]*domain.Auction, error) {
	m.ListCalls++
	m.ListArgs = append(m.ListArgs, ListArgs{Ctx: ctx})
	return m.ListResult, m.ListErr
}

type UpdateArgs struct {
	Ctx     context.Context
	Auction *domain.Auction
}

func (m *MockAuctionRepository) Update(ctx context.Context, auction *domain.Auction) error {
	m.UpdateCalls++
	m.UpdateArgs = append(m.UpdateArgs, UpdateArgs{Ctx: ctx, Auction: auction})
	return m.UpdateErr
}

type FindExpiredActiveArgs struct {
	Ctx context.Context
}

func (m *MockAuctionRepository) FindExpiredActive(ctx context.Context) ([]*domain.Auction, error) {
	m.FindExpiredActiveCalls++
	m.FindExpiredActiveArgs = append(m.FindExpiredActiveArgs, FindExpiredActiveArgs{Ctx: ctx})
	return m.FindExpiredActiveResult, m.FindExpiredActiveErr
}

type FindEndingSoonArgs struct {
	Ctx              context.Context
	ThresholdSeconds int
}

func (m *MockAuctionRepository) FindEndingSoon(ctx context.Context, thresholdSeconds int) ([]*domain.Auction, error) {
	m.FindEndingSoonCalls++
	m.FindEndingSoonArgs = append(m.FindEndingSoonArgs, FindEndingSoonArgs{Ctx: ctx, ThresholdSeconds: thresholdSeconds})
	return m.FindEndingSoonResult, m.FindEndingSoonErr
}

type MockEventPublisher struct {
	PublishAuctionCreatedCalls    int
	PublishAuctionCreatedArgs     []PublishAuctionCreatedArgs
	PublishAuctionCreatedErr      error
	PublishAuctionEndingSoonCalls int
	PublishAuctionEndingSoonArgs  []PublishAuctionEndingSoonArgs
	PublishAuctionEndingSoonErr   error
	PublishAuctionEndedCalls      int
	PublishAuctionEndedArgs       []PublishAuctionEndedArgs
	PublishAuctionEndedErr        error
	CloseCalls                    int
	CloseErr                      error
}

type PublishAuctionCreatedArgs struct {
	Ctx     context.Context
	Auction *domain.Auction
}

func (m *MockEventPublisher) PublishAuctionCreated(ctx context.Context, auction *domain.Auction) error {
	m.PublishAuctionCreatedCalls++
	m.PublishAuctionCreatedArgs = append(m.PublishAuctionCreatedArgs, PublishAuctionCreatedArgs{Ctx: ctx, Auction: auction})
	return m.PublishAuctionCreatedErr
}

type PublishAuctionEndingSoonArgs struct {
	Ctx     context.Context
	Auction *domain.Auction
}

func (m *MockEventPublisher) PublishAuctionEndingSoon(ctx context.Context, auction *domain.Auction) error {
	m.PublishAuctionEndingSoonCalls++
	m.PublishAuctionEndingSoonArgs = append(m.PublishAuctionEndingSoonArgs, PublishAuctionEndingSoonArgs{Ctx: ctx, Auction: auction})
	return m.PublishAuctionEndingSoonErr
}

type PublishAuctionEndedArgs struct {
	Ctx     context.Context
	Auction *domain.Auction
}

func (m *MockEventPublisher) PublishAuctionEnded(ctx context.Context, auction *domain.Auction) error {
	m.PublishAuctionEndedCalls++
	m.PublishAuctionEndedArgs = append(m.PublishAuctionEndedArgs, PublishAuctionEndedArgs{Ctx: ctx, Auction: auction})
	return m.PublishAuctionEndedErr
}

func (m *MockEventPublisher) Close() error {
	m.CloseCalls++
	return m.CloseErr
}
