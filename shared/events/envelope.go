package events

import (
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	EventID       string      `json:"event_id"`
	EventType     string      `json:"event_type"`
	EventVersion  string      `json:"event_version"`
	OccurredAt    time.Time   `json:"occurred_at"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Payload       interface{} `json:"payload"`
}

// NewEnvelope creates an Envelope with a generated UUID and the current UTC time.
func NewEnvelope(eventType, eventVersion, correlationID string, payload interface{}) Envelope {
	return Envelope{
		EventID:       uuid.New().String(),
		EventType:     eventType,
		EventVersion:  eventVersion,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Payload:       payload,
	}
}
