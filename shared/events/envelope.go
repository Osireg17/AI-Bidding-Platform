package events

import (
	"time"

	"github.com/google/uuid"
)

// === CONTEXT ===
// Purpose: Universal event envelope that wraps every domain event published to RabbitMQ.
// All services produce and consume events using this envelope for consistency and traceability.
//
// === DATA / STATE ===
// Envelope is created per-event, immutable after construction.
// Contains metadata (id, type, version, timestamp, correlation) + arbitrary payload.
//
// === BEHAVIOR: NewEnvelope ===
// Input: event type string, version string, correlation ID string, payload (any)
// Output: populated Envelope with generated UUID and current timestamp
// Logic:
//   GENERATE unique event ID (UUID)
//   CAPTURE current UTC timestamp
//   ASSIGN all fields into Envelope struct
//   RETURN the Envelope

// Envelope wraps every domain event with metadata for tracing and versioning.
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
