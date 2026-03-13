package broadcaster

import (
	"encoding/json"
	"sync"

	"github.com/Osireg17/AI-Bidding-Platform/services/bff/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const subscriberBufferSize = 32

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]chan domain.SSEEvent
	logger      *zap.Logger
}

func NewBroadcaster(logger *zap.Logger) *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string]chan domain.SSEEvent),
		logger:      logger,
	}
}

func (b *Broadcaster) Subscribe() (<-chan domain.SSEEvent, func()) {

	key := uuid.NewString()
	ch := make(chan domain.SSEEvent, subscriberBufferSize)

	b.mu.Lock()
	b.subscribers[key] = ch
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		if _, exists := b.subscribers[key]; exists {
			delete(b.subscribers, key)
			close(ch)
		}
		b.mu.Unlock()
	}

	return ch, unsubscribe
}

func (b *Broadcaster) Broadcast(eventName string, payload any) {

	marshal, err := json.Marshal(payload)
	if err != nil {
		b.logger.Error("failed to marshal broadcast payload", zap.String("eventName", eventName), zap.Error(err))
		return
	}

	event := domain.SSEEvent{Name: eventName, Payload: marshal}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for key, ch := range b.subscribers {
		select {
		case ch <- event:
			// sent successfully
		default:
			b.logger.Warn("subscriber channel full, dropping event", zap.String("key", key), zap.String("eventName", eventName))
		}
	}

}

var _ domain.EventBroadcaster = (*Broadcaster)(nil)
