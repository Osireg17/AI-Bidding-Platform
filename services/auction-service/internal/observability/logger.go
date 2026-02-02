package observability

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// === CONTEXT ===
// Purpose: Create a structured zap logger tagged with the service name.
// Every log line from this service will include "service": "auction-service".
//
// === BEHAVIOR: NewLogger ===
// Input: log level string ("debug", "info", "warn", "error")
// Output: *zap.Logger configured for production with service name field, or error
// Logic:
//   PARSE the level string into a zapcore.Level
//   BUILD a production config with the parsed level
//   ADD "service" field to every log entry
//   RETURN the configured logger

// NewLogger creates a production zap logger with the given level and service name.
func NewLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger.With(zap.String("service", "auction-service")), nil
}
