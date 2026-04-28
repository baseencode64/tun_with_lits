package client

import (
	"log/slog"
)

// SlogAdapter adapts slog.Logger to our Logger interface
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new adapter from slog.Logger
func NewSlogAdapter(logger *slog.Logger) Logger {
	if logger == nil {
		return &noopLogger{}
	}
	return &SlogAdapter{logger: logger}
}

// Debug logs debug messages
func (a *SlogAdapter) Debug(msg string, keysAndValues ...interface{}) {
	a.logger.Debug(msg, keysAndValues...)
}

// Info logs info messages
func (a *SlogAdapter) Info(msg string, keysAndValues ...interface{}) {
	a.logger.Info(msg, keysAndValues...)
}

// Error logs error messages
func (a *SlogAdapter) Error(msg string, keysAndValues ...interface{}) {
	a.logger.Error(msg, keysAndValues...)
}
