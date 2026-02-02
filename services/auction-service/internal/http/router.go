package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// === CONTEXT ===
// Purpose: Gin router setup — middleware registration, route mapping, health endpoint.
// Single place for all routing configuration.
//
// === DEPENDENCIES ===
// gin — HTTP framework
// AuctionHandler — handler methods to register
// zap.Logger — for request logging middleware

// === BEHAVIOR: NewRouter ===
// Input: AuctionHandler, *zap.Logger
// Output: configured *gin.Engine
// Logic:
//   CREATE gin engine with recovery middleware
//   ADD correlation ID middleware (generates UUID if not present in header)
//   ADD request logging middleware
//   REGISTER health check endpoint
//   REGISTER auction API routes
//   RETURN engine

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

// correlationIDMiddleware adds a correlation ID to every request.
// If the X-Correlation-ID header is present, it uses that; otherwise it generates a new UUID.
func correlationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}

// requestLoggingMiddleware logs each request with method, path, status, and latency.
func requestLoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("correlation_id", c.GetString("correlation_id")),
		)
	}
}
