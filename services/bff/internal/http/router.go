package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(handler *BFFHandler, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(correlationIDMiddleware())
	r.Use(requestLoggingMiddleware(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.GET("/state", handler.HandleGetState)
		api.GET("/stream", handler.HandleStream)
	}

	return r
}
