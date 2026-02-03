package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewAuction(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		duration := 2 * time.Minute
		auction, err := NewAuction("Rare Coin", "A shiny coin", 100.0, duration)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Title != "Rare Coin" {
			t.Fatalf("expected title to be set")
		}
		if auction.Description != "A shiny coin" {
			t.Fatalf("expected description to be set")
		}
		if auction.StartPrice != 100.0 {
			t.Fatalf("expected start price to be set")
		}
		if auction.CurrentPrice != 100.0 {
			t.Fatalf("expected current price to equal start price")
		}
		if auction.Status != StatusPending {
			t.Fatalf("expected status pending")
		}
		if auction.StartTime.IsZero() || auction.EndTime.IsZero() {
			t.Fatalf("expected start and end times to be set")
		}
		if !auction.EndTime.After(auction.StartTime) {
			t.Fatalf("expected end time after start time")
		}
		if auction.CreatedAt.IsZero() || auction.UpdatedAt.IsZero() {
			t.Fatalf("expected created/updated times to be set")
		}
	})

	cases := []struct {
		name      string
		title     string
		price     float64
		duration  time.Duration
		wantError error
	}{
		{"empty title", "", 100, time.Minute, ErrInvalidAuctionData},
		{"zero price", "x", 0, time.Minute, ErrInvalidAuctionData},
		{"negative price", "x", -1, time.Minute, ErrInvalidAuctionData},
		{"zero duration", "x", 1, 0, ErrInvalidAuctionData},
		{"negative duration", "x", 1, -time.Second, ErrInvalidAuctionData},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAuction(tc.title, "desc", tc.price, tc.duration)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, tc.wantError) {
				t.Fatalf("expected error %v, got %v", tc.wantError, err)
			}
		})
	}
}

func TestActivate(t *testing.T) {
	t.Run("from pending", func(t *testing.T) {
		auction := &Auction{Status: StatusPending, UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
		err := auction.Activate()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Status != StatusActive {
			t.Fatalf("expected status active")
		}
		if !auction.UpdatedAt.After(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("expected updated_at to change")
		}
	})

	cases := []struct {
		name   string
		status AuctionStatus
	}{
		{"from active", StatusActive},
		{"from ending_soon", StatusEndingSoon},
		{"from closed", StatusClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auction := &Auction{Status: tc.status}
			err := auction.Activate()
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrInvalidStatus) {
				t.Fatalf("expected ErrInvalidStatus, got %v", err)
			}
		})
	}
}

func TestMarkEndingSoon(t *testing.T) {
	t.Run("from active", func(t *testing.T) {
		auction := &Auction{Status: StatusActive, UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
		err := auction.MarkEndingSoon()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Status != StatusEndingSoon {
			t.Fatalf("expected status ending_soon")
		}
		if !auction.UpdatedAt.After(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("expected updated_at to change")
		}
	})

	cases := []struct {
		name   string
		status AuctionStatus
	}{
		{"from pending", StatusPending},
		{"from ending_soon", StatusEndingSoon},
		{"from closed", StatusClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auction := &Auction{Status: tc.status}
			err := auction.MarkEndingSoon()
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrInvalidStatus) {
				t.Fatalf("expected ErrInvalidStatus, got %v", err)
			}
		})
	}
}

func TestClose(t *testing.T) {
	t.Run("from active with winner", func(t *testing.T) {
		auction := &Auction{
			Status:       StatusActive,
			CurrentPrice: 50,
			UpdatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		err := auction.Close(42, 75)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Status != StatusClosed {
			t.Fatalf("expected status closed")
		}
		if auction.WinnerBotID != 42 {
			t.Fatalf("expected winner bot id set")
		}
		if auction.CurrentPrice != 75 {
			t.Fatalf("expected current price updated to winning bid")
		}
		if !auction.UpdatedAt.After(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("expected updated_at to change")
		}
	})

	t.Run("from ending soon with winner", func(t *testing.T) {
		auction := &Auction{Status: StatusEndingSoon, CurrentPrice: 20}
		err := auction.Close(7, 25)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Status != StatusClosed {
			t.Fatalf("expected status closed")
		}
		if auction.WinnerBotID != 7 {
			t.Fatalf("expected winner bot id set")
		}
		if auction.CurrentPrice != 25 {
			t.Fatalf("expected current price updated to winning bid")
		}
	})

	t.Run("no winner keeps current price", func(t *testing.T) {
		auction := &Auction{Status: StatusActive, CurrentPrice: 60}
		err := auction.Close(0, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if auction.Status != StatusClosed {
			t.Fatalf("expected status closed")
		}
		if auction.WinnerBotID != 0 {
			t.Fatalf("expected winner bot id zero")
		}
		if auction.CurrentPrice != 60 {
			t.Fatalf("expected current price unchanged")
		}
	})

	cases := []struct {
		name   string
		status AuctionStatus
	}{
		{"from pending", StatusPending},
		{"from closed", StatusClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auction := &Auction{Status: tc.status}
			err := auction.Close(1, 10)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !errors.Is(err, ErrInvalidStatus) {
				t.Fatalf("expected ErrInvalidStatus, got %v", err)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		status   AuctionStatus
		endTime  time.Time
		expected bool
	}{
		{"active + past", StatusActive, past, true},
		{"active + future", StatusActive, future, false},
		{"pending + past", StatusPending, past, false},
		{"closed + past", StatusClosed, past, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auction := &Auction{Status: tc.status, EndTime: tc.endTime}
			if auction.IsExpired() != tc.expected {
				t.Fatalf("expected %v", tc.expected)
			}
		})
	}
}

func TestIsEndingSoon(t *testing.T) {
	within := time.Now().UTC().Add(10 * time.Second)
	outside := time.Now().UTC().Add(2 * time.Minute)
	threshold := 30 * time.Second

	cases := []struct {
		name     string
		status   AuctionStatus
		endTime  time.Time
		expected bool
	}{
		{"active + within threshold", StatusActive, within, true},
		{"active + outside threshold", StatusActive, outside, false},
		{"pending + within threshold", StatusPending, within, false},
		{"closed + within threshold", StatusClosed, within, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auction := &Auction{Status: tc.status, EndTime: tc.endTime}
			if auction.IsEndingSoon(threshold) != tc.expected {
				t.Fatalf("expected %v", tc.expected)
			}
		})
	}
}
