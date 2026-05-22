package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
	"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BankingHandler struct {
	svc    *service.BankingService
	logger *zap.Logger
}

func NewBankingHandler(svc *service.BankingService, logger *zap.Logger) *BankingHandler {
	return &BankingHandler{svc: svc, logger: logger}
}

func (h *BankingHandler) GetWallet(c *gin.Context) {
	botIDParam := c.Param("bot_id")
	botID, err := strconv.ParseInt(botIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot_id"})
		return
	}

	wallet, err := h.svc.GetWallet(c.Request.Context(), botID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		h.logger.Error("failed to get wallet", zap.Int64("bot_id", botID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func (h *BankingHandler) Buyout(c *gin.Context) {
	itemIDParam := c.Param("item_id")
	itemID, err := strconv.ParseInt(itemIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item_id"})
		return
	}

	newBalance, err := h.svc.Buyout(c.Request.Context(), itemID)
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("failed to process buyout", zap.Int64("item_id", itemID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"new_balance": newBalance})
}
