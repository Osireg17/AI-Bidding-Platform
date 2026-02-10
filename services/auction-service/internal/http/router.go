package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the Gin engine with middleware and routes.
func NewRouter(handler *AuctionHandler, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Recovery middleware — catches panics and returns 500.
	r.Use(gin.Recovery())

	// Correlation ID middleware — ensures every request has a traceable ID.
	r.Use(correlationIDMiddleware())

	// Request logging middleware.
	r.Use(requestLoggingMiddleware(logger))

	// Health check.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auction API routes.
	api := r.Group("/api")
	{
		api.POST("/auctions", handler.HandleCreateAuction)
		api.GET("/auctions", handler.HandleListAuctions)
		api.GET("/auctions/:id", handler.HandleGetAuction)
	}

	return r
}
