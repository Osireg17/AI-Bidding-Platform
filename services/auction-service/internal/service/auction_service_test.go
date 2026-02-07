package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	testutil "github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/test"
)

func TestCreateAuction_HappyPath(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	title := "Auction 1"
	description := "Test auction"
	startPrice := 25.0
	duration := 10 * time.Minute

	before := time.Now().UTC()
	auction, err := svc.CreateAuction(ctx, title, description, startPrice, duration)
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auction == nil {
		t.Fatalf("expected auction, got nil")
	}

	if auction.Title != title {
		t.Fatalf("expected title %q, got %q", title, auction.Title)
	}
	if auction.Description != description {
		t.Fatalf("expected description %q, got %q", description, auction.Description)
	}
	if auction.StartPrice != startPrice {
		t.Fatalf("expected start price %.2f, got %.2f", startPrice, auction.StartPrice)
	}
	if auction.CurrentPrice != startPrice {
		t.Fatalf("expected current price %.2f, got %.2f", startPrice, auction.CurrentPrice)
	}
	if auction.Status != domain.StatusActive {
		t.Fatalf("expected status %q, got %q", domain.StatusActive, auction.Status)
	}
	if auction.StartTime.Before(before) || auction.StartTime.After(after) {
		t.Fatalf("expected StartTime within test window, got %v", auction.StartTime)
	}
	if !auction.EndTime.After(auction.StartTime) {
		t.Fatalf("expected EndTime after StartTime")
	}

	if repo.CreateCalls != 1 {
		t.Fatalf("expected repo.Create called once, got %d", repo.CreateCalls)
	}
	if repo.CreateArgs[0].Auction != auction {
		t.Fatalf("expected repo.Create to receive created auction")
	}

	if publisher.PublishAuctionCreatedCalls != 1 {
		t.Fatalf("expected PublishAuctionCreated called once, got %d", publisher.PublishAuctionCreatedCalls)
	}
	if publisher.PublishAuctionCreatedArgs[0].Auction != auction {
		t.Fatalf("expected PublishAuctionCreated to receive created auction")
	}
}

func TestCreateAuction_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		CreateErr: context.DeadlineExceeded,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	_, err := svc.CreateAuction(ctx, "Title", "Description", 10.0, 5*time.Minute)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded error, got %v", err)
	}

	if repo.CreateCalls != 1 {
		t.Fatalf("expected repo.Create called once, got %d", repo.CreateCalls)
	}

	if publisher.PublishAuctionCreatedCalls != 0 {
		t.Fatalf("expected PublishAuctionCreated not called, got %d", publisher.PublishAuctionCreatedCalls)
	}
}

func TestCreateAuction_InvalidInput(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	_, err := svc.CreateAuction(ctx, "", "Description", 10.0, 5*time.Minute)
	if err == nil {
		t.Fatalf("expected error for empty title, got nil")
	}

	_, err = svc.CreateAuction(ctx, "Title", "Description", -5.0, 5*time.Minute)
	if err == nil {
		t.Fatalf("expected error for negative start price, got nil")
	}

	_, err = svc.CreateAuction(ctx, "Title", "Description", 10.0, -1*time.Minute)
	if err == nil {
		t.Fatalf("expected error for negative duration, got nil")
	}

	if repo.CreateCalls != 0 {
		t.Fatalf("expected repo.Create not called, got %d", repo.CreateCalls)
	}

	if publisher.PublishAuctionCreatedCalls != 0 {
		t.Fatalf("expected PublishAuctionCreated not called, got %d", publisher.PublishAuctionCreatedCalls)
	}
}

func TestGetAuction(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		GetByIDResult: &domain.Auction{
			ID:           1,
			Title:        "Test Auction",
			Description:  "A test auction",
			StartPrice:   10.0,
			CurrentPrice: 10.0,
			Status:       domain.StatusActive,
			StartTime:    time.Now().Add(-time.Hour),
			EndTime:      time.Now().Add(time.Hour),
		},
		GetByIDErr: nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	auction, err := svc.GetAuction(ctx, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auction == nil {
		t.Fatalf("expected auction, got nil")
	}
	if auction.ID != 1 {
		t.Fatalf("expected auction ID 1, got %d", auction.ID)
	}

	repo.GetByIDResult = nil
	repo.GetByIDErr = domain.ErrAuctionNotFound

	_, err = svc.GetAuction(ctx, 999)
	if err == nil {
		t.Fatalf("expected error for non-existent auction, got nil")
	}
	if !errors.Is(err, domain.ErrAuctionNotFound) {
		t.Fatalf("expected ErrAuctionNotFound, got %v", err)
	}
}

func TestGetAuction_NilResult(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		GetByIDResult: nil,
		GetByIDErr:    nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	auction, err := svc.GetAuction(ctx, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auction != nil {
		t.Fatalf("expected nil auction, got %v", auction)
	}
}

func TestListAuctions(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		ListResult: []*domain.Auction{
			{
				ID:           1,
				Title:        "Auction 1",
				Description:  "First auction",
				StartPrice:   10.0,
				CurrentPrice: 10.0,
				Status:       domain.StatusActive,
				StartTime:    time.Now().Add(-time.Hour),
				EndTime:      time.Now().Add(time.Hour),
			},
			{
				ID:           2,
				Title:        "Auction 2",
				Description:  "Second auction",
				StartPrice:   20.0,
				CurrentPrice: 20.0,
				Status:       domain.StatusActive,
				StartTime:    time.Now().Add(-2 * time.Hour),
				EndTime:      time.Now().Add(2 * time.Hour),
			},
		},
		ListErr: nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	auctions, err := svc.ListAuctions(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auctions) != 2 {
		t.Fatalf("expected 2 auctions, got %d", len(auctions))
	}
	if auctions[0].ID != 1 || auctions[1].ID != 2 {
		t.Fatalf("unexpected auction IDs in result")
	}

	repo.ListResult = nil
	repo.ListErr = errors.New("database error")

	_, err = svc.ListAuctions(ctx)
	if err == nil {
		t.Fatalf("expected error from repo.List, got nil")
	}
	if !errors.Is(err, repo.ListErr) {
		t.Fatalf("expected error %v, got %v", repo.ListErr, err)
	}
}

func TestListAuctions_Empty(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		ListResult: []*domain.Auction{},
		ListErr:    nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	auctions, err := svc.ListAuctions(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auctions) != 0 {
		t.Fatalf("expected 0 auctions, got %d", len(auctions))
	}
}

func TestListAuctions_MultipleCalls(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		ListResult: []*domain.Auction{
			{ID: 1, Title: "Auction 1"},
			{ID: 2, Title: "Auction 2"},
		},
		ListErr: nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	for i := range [3]int{} {
		auctions, err := svc.ListAuctions(ctx)
		if err != nil {
			t.Fatalf("iteration %d: expected no error, got %v", i, err)
		}
		if len(auctions) != 2 {
			t.Fatalf("iteration %d: expected 2 auctions, got %d", i, len(auctions))
		}
	}

	if repo.ListCalls != 3 {
		t.Fatalf("expected repo.List called 3 times, got %d", repo.ListCalls)
	}
}

func TestProcessExpiredAuctions(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		FindExpiredActiveResult: []*domain.Auction{
			{
				ID:           1,
				Title:        "Expired Auction",
				Description:  "An expired auction",
				StartPrice:   10.0,
				CurrentPrice: 10.0,
				Status:       domain.StatusActive,
				StartTime:    time.Now().Add(-2 * time.Hour),
				EndTime:      time.Now().Add(-time.Hour),
			},
		},
		UpdateErr: nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	err := svc.ProcessExpiredAuctions(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.FindExpiredActiveCalls != 1 {
		t.Fatalf("expected repo.FindExpiredActive called once, got %d", repo.FindExpiredActiveCalls)
	}
	if repo.UpdateCalls != 1 {
		t.Fatalf("expected repo.Update called once, got %d", repo.UpdateCalls)
	}
	if repo.UpdateArgs[0].Auction.ID != 1 {
		t.Fatalf("expected repo.Update to receive auction ID 1, got %d", repo.UpdateArgs[0].Auction.ID)
	}

	if publisher.PublishAuctionEndedCalls != 1 {
		t.Fatalf("expected PublishAuctionEnded called once, got %d", publisher.PublishAuctionEndedCalls)
	}
	if publisher.PublishAuctionEndedArgs[0].Auction.ID != 1 {
		t.Fatalf("expected PublishAuctionEnded to receive auction ID 1, got %d", publisher.PublishAuctionEndedArgs[0].Auction.ID)
	}
}

func TestProcessExpiredAuctions_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		FindExpiredActiveResult: nil,
		FindExpiredActiveErr:    errors.New("database error"),
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	err := svc.ProcessExpiredAuctions(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, repo.FindExpiredActiveErr) {
		t.Fatalf("expected error %v, got %v", repo.FindExpiredActiveErr, err)
	}

	if repo.FindExpiredActiveCalls != 1 {
		t.Fatalf("expected repo.FindExpiredActive called once, got %d", repo.FindExpiredActiveCalls)
	}
	if repo.UpdateCalls != 0 {
		t.Fatalf("expected repo.Update not called, got %d", repo.UpdateCalls)
	}
	if publisher.PublishAuctionEndedCalls != 0 {
		t.Fatalf("expected PublishAuctionEnded not called, got %d", publisher.PublishAuctionEndedCalls)
	}
}

func TestProcessEndingSoonAuctions(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		FindEndingSoonResult: []*domain.Auction{
			{
				ID:           1,
				Title:        "Ending Soon Auction",
				Description:  "An auction ending soon",
				StartPrice:   10.0,
				CurrentPrice: 10.0,
				Status:       domain.StatusActive,
				StartTime:    time.Now().Add(-time.Hour),
				EndTime:      time.Now().Add(30 * time.Second),
			},
		},
		UpdateErr: nil,
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	err := svc.ProcessEndingSoonAuctions(ctx, 60)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.FindEndingSoonCalls != 1 {
		t.Fatalf("expected repo.FindEndingSoon called once, got %d", repo.FindEndingSoonCalls)
	}
	if repo.FindEndingSoonArgs[0].ThresholdSeconds != 60 {
		t.Fatalf("expected FindEndingSoon called with threshold 60, got %d", repo.FindEndingSoonArgs[0].ThresholdSeconds)
	}
	if repo.UpdateCalls != 1 {
		t.Fatalf("expected repo.Update called once, got %d", repo.UpdateCalls)
	}
	if repo.UpdateArgs[0].Auction.ID != 1 {
		t.Fatalf("expected repo.Update to receive auction ID 1, got %d", repo.UpdateArgs[0].Auction.ID)
	}
	if repo.UpdateArgs[0].Auction.Status != domain.StatusEndingSoon {
		t.Fatalf("expected repo.Update auction status %q, got %q", domain.StatusEndingSoon, repo.UpdateArgs[0].Auction.Status)
	}
	if publisher.PublishAuctionEndingSoonCalls != 1 {
		t.Fatalf("expected PublishAuctionEndingSoon called once, got %d", publisher.PublishAuctionEndingSoonCalls)
	}
	if publisher.PublishAuctionEndingSoonArgs[0].Auction.ID != 1 {
		t.Fatalf("expected PublishAuctionEndingSoon to receive auction ID 1, got %d", publisher.PublishAuctionEndingSoonArgs[0].Auction.ID)
	}
}

func TestProcessEndingSoonAuctions_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		FindEndingSoonResult: nil,
		FindEndingSoonErr:    errors.New("database error"),
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	err := svc.ProcessEndingSoonAuctions(ctx, 60)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, repo.FindEndingSoonErr) {
		t.Fatalf("expected error %v, got %v", repo.FindEndingSoonErr, err)
	}

	if repo.FindEndingSoonCalls != 1 {
		t.Fatalf("expected repo.FindEndingSoon called once, got %d", repo.FindEndingSoonCalls)
	}
	if repo.UpdateCalls != 0 {
		t.Fatalf("expected repo.Update not called, got %d", repo.UpdateCalls)
	}
	if publisher.PublishAuctionEndingSoonCalls != 0 {
		t.Fatalf("expected PublishAuctionEndingSoon not called, got %d", publisher.PublishAuctionEndingSoonCalls)
	}
}

func TestProcessEndingSoonAuctions_UpdateError(t *testing.T) {
	ctx := context.Background()
	repo := &testutil.MockAuctionRepository{
		FindEndingSoonResult: []*domain.Auction{
			{
				ID:           1,
				Title:        "Ending Soon Auction",
				Description:  "An auction ending soon",
				StartPrice:   10.0,
				CurrentPrice: 10.0,
				Status:       domain.StatusActive,
				StartTime:    time.Now().Add(-time.Hour),
				EndTime:      time.Now().Add(30 * time.Second),
			},
		},
		UpdateErr: errors.New("update failed"),
	}
	publisher := &testutil.MockEventPublisher{}
	logger := testutil.NewTestLogger(t)

	svc := NewAuctionService(repo, publisher, logger)

	err := svc.ProcessEndingSoonAuctions(ctx, 60)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.FindEndingSoonCalls != 1 {
		t.Fatalf("expected repo.FindEndingSoon called once, got %d", repo.FindEndingSoonCalls)
	}
	if repo.UpdateCalls != 1 {
		t.Fatalf("expected repo.Update called once, got %d", repo.UpdateCalls)
	}
	if publisher.PublishAuctionEndingSoonCalls != 0 {
		t.Fatalf("expected PublishAuctionEndingSoon not called, got %d", publisher.PublishAuctionEndingSoonCalls)
	}
}
