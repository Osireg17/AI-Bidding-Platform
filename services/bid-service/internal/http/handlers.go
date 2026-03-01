package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BidHandler struct {
	svc    *service.BidService
	logger *zap.Logger
}

func NewBidHandler(svc *service.BidService, logger *zap.Logger) *BidHandler {
	return &BidHandler{svc: svc, logger: logger}
}

type createBidRequest struct {
	AuctionID int64   `json:"auction_id" binding:"required"`
	BotID     int64   `json:"bot_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
}

type bidResponse struct {
	ID        int64   `json:"id"`
	AuctionID int64   `json:"auction_id"`
	BotID     int64   `json:"bot_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

func toBidResponse(b *domain.Bid) bidResponse {
	return bidResponse{
		ID:        b.ID,
		AuctionID: b.AuctionID,
		BotID:     b.BotID,
		Amount:    b.Amount,
		Status:    string(b.Status),
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
	}
}

func (h *BidHandler) HandlePlaceBid(c *gin.Context) {
	var req createBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bid, err := h.svc.PlaceBid(c.Request.Context(), req.AuctionID, req.BotID, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidBidData):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrAuctionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrBidTooLow), errors.Is(err, domain.ErrAuctionNotActive):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrLockAcquisitionFailed):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		default:
			h.logger.Error("failed to place bid",
				zap.Int64("auction_id", req.AuctionID),
				zap.Int64("bot_id", req.BotID),
				zap.Float64("amount", req.Amount),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, toBidResponse(bid))
}
