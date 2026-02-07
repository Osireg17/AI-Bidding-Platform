package events

import (
	"reflect"
	"testing"
	"time"
)

func TestNewEnvelope_PopulatesFields(t *testing.T) {
	eventType := "bid.created"
	eventVersion := "v1"
	correlationID := "corr-123"
	payload := map[string]string{"k": "v"}

	before := time.Now().UTC()
	env := NewEnvelope(eventType, eventVersion, correlationID, payload)
	after := time.Now().UTC()

	if env.EventID == "" {
		t.Fatalf("expected EventID to be set")
	}
	if env.EventType != eventType {
		t.Fatalf("expected EventType %q, got %q", eventType, env.EventType)
	}
	if env.EventVersion != eventVersion {
		t.Fatalf("expected EventVersion %q, got %q", eventVersion, env.EventVersion)
	}
	if env.CorrelationID != correlationID {
		t.Fatalf("expected CorrelationID %q, got %q", correlationID, env.CorrelationID)
	}
	if !reflect.DeepEqual(env.Payload, payload) {
		t.Fatalf("expected Payload to match input")
	}
	if env.OccurredAt.Before(before) || env.OccurredAt.After(after) {
		t.Fatalf("expected OccurredAt within test window, got %v", env.OccurredAt)
	}
}

func TestNewEnvelope_GeneratesUUID(t *testing.T) {
	env1 := NewEnvelope("event.type", "v1", "corr", nil)
	env2 := NewEnvelope("event.type", "v1", "corr", nil)

	if env1.EventID == "" || env2.EventID == "" {
		t.Fatalf("expected EventID to be set")
	}
	if env1.EventID == env2.EventID {
		t.Fatalf("expected different EventIDs, got the same %q", env1.EventID)
	}
}

func TestNewEnvelope_WithCorrelationID(t *testing.T) {
	correlationID := "corr-123"
	env := NewEnvelope("event.type", "v1", correlationID, nil)

	if env.CorrelationID != correlationID {
		t.Fatalf("expected CorrelationID %q, got %q", correlationID, env.CorrelationID)
	}
}

func TestNewEnvelope_OmitsCorrelationIDWhenEmpty(t *testing.T) {
	env := NewEnvelope("event.type", "v1", "", "payload")
	if env.CorrelationID != "" {
		t.Fatalf("expected empty CorrelationID, got %q", env.CorrelationID)
	}
}

func TestNewEnvelope_UtcTime(t *testing.T) {
	env := NewEnvelope("event.type", "v1", "corr", nil)

	if env.OccurredAt.Location() != time.UTC {
		t.Fatalf("expected OccurredAt in UTC, got %v", env.OccurredAt.Location())
	}
}
