package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/auction-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuctionHandler struct {
	svc    *service.AuctionService
	logger *zap.Logger
}

// NewAuctionHandler creates an AuctionHandler with its dependencies.
func NewAuctionHandler(svc *service.AuctionService, logger *zap.Logger) *AuctionHandler {
	return &AuctionHandler{svc: svc, logger: logger}
}

// createAuctionRequest is the expected JSON body for POST /api/auctions.
type createAuctionRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	StartPrice  float64 `json:"start_price" binding:"required,gt=0"`
	DurationSec int     `json:"duration_sec" binding:"required,gt=0"`
}

// auctionResponse is the JSON representation returned to clients.
type auctionResponse struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	StartPrice   float64 `json:"start_price"`
	CurrentPrice float64 `json:"current_price"`
	Status       string  `json:"status"`
	WinnerBotID  int64   `json:"winner_bot_id,omitempty"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toAuctionResponse(a *domain.Auction) auctionResponse {
	return auctionResponse{
		ID:           a.ID,
		Title:        a.Title,
		Description:  a.Description,
		StartPrice:   a.StartPrice,
		CurrentPrice: a.CurrentPrice,
		Status:       string(a.Status),
		WinnerBotID:  a.WinnerBotID,
		StartTime:    a.StartTime.Format(time.RFC3339),
		EndTime:      a.EndTime.Format(time.RFC3339),
		CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AuctionHandler) HandleCreateAuction(c *gin.Context) {
	var req createAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	duration := time.Duration(req.DurationSec) * time.Second
	auction, err := h.svc.CreateAuction(c.Request.Context(), req.Title, req.Description, req.StartPrice, duration)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAuctionData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("failed to create auction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, toAuctionResponse(auction))
}

func (h *AuctionHandler) HandleGetAuction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid auction id"})
		return
	}

	auction, err := h.svc.GetAuction(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrAuctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "auction not found"})
			return
		}
		h.logger.Error("failed to get auction", zap.Int64("auction_id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toAuctionResponse(auction))
}

func (h *AuctionHandler) HandleListAuctions(c *gin.Context) {
	auctions, err := h.svc.ListAuctions(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list auctions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := make([]auctionResponse, 0, len(auctions))
	for _, a := range auctions {
		response = append(response, toAuctionResponse(a))
	}
	c.JSON(http.StatusOK, response)
}
