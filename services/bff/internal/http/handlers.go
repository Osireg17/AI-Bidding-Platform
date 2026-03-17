package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BFFHandler struct {
	store       domain.StateStore
	broadcaster domain.EventBroadcaster
	logger      *zap.Logger
}

func NewBFFHandler(store domain.StateStore, broadcaster domain.EventBroadcaster, logger *zap.Logger) *BFFHandler {
	return &BFFHandler{store: store, broadcaster: broadcaster, logger: logger}
}

// stateResponse is the JSON shape for GET /api/state.
type stateResponse struct {
	Auction *auctionJSON `json:"auction"`
	Bids    []bidJSON    `json:"bids"`
	Winner  *winnerJSON  `json:"winner"`
}

type auctionJSON struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	StartPrice   float64 `json:"start_price"`
	CurrentPrice float64 `json:"current_price"`
	Status       string  `json:"status"`
	EndTime      string  `json:"end_time"`
}

type bidJSON struct {
	BotName   string  `json:"bot_name"`
	BotID     int64   `json:"bot_id"`
	Amount    float64 `json:"amount"`
	Timestamp string  `json:"timestamp"`
}

type winnerJSON struct {
	BotName     string  `json:"bot_name"`
	BotID       int64   `json:"bot_id"`
	Amount      float64 `json:"amount"`
	FinalStatus string  `json:"final_status"`
}

func toStateResponse(s domain.AuctionState) stateResponse {
	resp := stateResponse{
		Bids: make([]bidJSON, 0, len(s.Bids)),
	}

	if s.HasAuction {
		resp.Auction = &auctionJSON{
			ID:           s.Auction.ID,
			Title:        s.Auction.Title,
			Description:  s.Auction.Description,
			StartPrice:   s.Auction.StartPrice,
			CurrentPrice: s.Auction.CurrentPrice,
			Status:       s.Auction.Status,
			EndTime:      s.Auction.EndTime.Format(time.RFC3339),
		}
	}

	for _, b := range s.Bids {
		resp.Bids = append(resp.Bids, bidJSON{
			BotName:   b.BotName,
			BotID:     b.BotID,
			Amount:    b.Amount,
			Timestamp: b.Timestamp.Format(time.RFC3339),
		})
	}

	if s.HasWinner {
		resp.Winner = &winnerJSON{
			BotName:     s.Winner.BotName,
			BotID:       s.Winner.BotID,
			Amount:      s.Winner.Amount,
			FinalStatus: s.Winner.FinalStatus,
		}
	}

	return resp
}

// HandleGetState returns a JSON snapshot of the current auction state.
func (h *BFFHandler) HandleGetState(c *gin.Context) {
	state := h.store.GetState()
	c.JSON(http.StatusOK, toStateResponse(state))
}

// HandleStream upgrades the connection to SSE and streams events until the client disconnects.
func (h *BFFHandler) HandleStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch, unsubscribe := h.broadcaster.Subscribe()
	defer unsubscribe()

	// Send an immediate snapshot so the client has state on connect.
	state := h.store.GetState()
	if b, err := json.Marshal(toStateResponse(state)); err == nil {
		fmt.Fprintf(c.Writer, "event: auction.snapshot\ndata: %s\n\n", b)
		c.Writer.Flush()
	}

	ctx := c.Request.Context()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Name, event.Payload)
			c.Writer.Flush()
		case <-ctx.Done():
			h.logger.Info("SSE client disconnected",
				zap.String("remote_addr", c.Request.RemoteAddr),
			)
			return
		}
	}
}
