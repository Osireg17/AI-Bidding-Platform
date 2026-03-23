package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRouter_Health(t *testing.T) {
	store := &fakeStore{}
	broadcaster := newFakeBroadcaster()
	close(broadcaster.ch)

	router := NewRouter(newTestHandler(store, broadcaster), zap.NewNop(), "*")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestRouter_StateRoute(t *testing.T) {
	store := &fakeStore{}
	broadcaster := newFakeBroadcaster()

	router := NewRouter(newTestHandler(store, broadcaster), zap.NewNop(), "*")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/state", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_CorrelationIDMiddleware_GeneratesID(t *testing.T) {
	router := NewRouter(newTestHandler(&fakeStore{}, newFakeBroadcaster()), zap.NewNop(), "*")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
}

func TestRouter_CorrelationIDMiddleware_EchoesExistingID(t *testing.T) {
	router := NewRouter(newTestHandler(&fakeStore{}, newFakeBroadcaster()), zap.NewNop(), "*")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Correlation-ID", "test-id-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, "test-id-123", w.Header().Get("X-Correlation-ID"))
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter(newTestHandler(&fakeStore{}, newFakeBroadcaster()), zap.NewNop(), "*")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/unknown", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
