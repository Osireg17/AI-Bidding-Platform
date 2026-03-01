package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCorrelationIDMiddleware_GeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(correlationIDMiddleware())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(recorder, request)

	correlationID := recorder.Header().Get("X-Correlation-ID")
	if correlationID == "" {
		t.Fatalf("expected X-Correlation-ID to be set")
	}
	if _, err := uuid.Parse(correlationID); err != nil {
		t.Fatalf("expected valid correlation ID, got %q", correlationID)
	}
}

func TestCorrelationIDMiddleware_PreservesHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(correlationIDMiddleware())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Correlation-ID", "request-123")

	router.ServeHTTP(recorder, request)

	correlationID := recorder.Header().Get("X-Correlation-ID")
	if correlationID != "request-123" {
		t.Fatalf("expected correlation ID request-123, got %q", correlationID)
	}
}

func TestRequestLoggingMiddleware_LogsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	router := gin.New()
	router.Use(requestLoggingMiddleware(logger))
	router.GET("/ping", func(c *gin.Context) {
		c.Set("correlation_id", "cid-42")
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Message != "request" {
		t.Fatalf("expected log message 'request', got %q", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["method"] != http.MethodGet {
		t.Fatalf("expected method %q, got %v", http.MethodGet, fields["method"])
	}
	if fields["path"] != "/ping" {
		t.Fatalf("expected path /ping, got %v", fields["path"])
	}
	if fields["status"] != int64(http.StatusNoContent) {
		t.Fatalf("expected status %d, got %v", http.StatusNoContent, fields["status"])
	}
	if fields["correlation_id"] != "cid-42" {
		t.Fatalf("expected correlation_id cid-42, got %v", fields["correlation_id"])
	}
}
