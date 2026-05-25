package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

// ErrReconnectionStopped is returned when the reconnector is stopped via Stop().
var ErrReconnectionStopped = errors.New("reconnection stopped")

// Default reconnection settings
const (
	DefaultMinBackoff    = 5 * time.Second
	DefaultMaxBackoff    = 5 * time.Minute
	DefaultBackoffFactor = 2.0
	DefaultMaxRetries    = 0 // 0 = unlimited retries
)

// ReconnectionConfig holds settings for reconnection with exponential backoff.
type ReconnectionConfig struct {
	// MaxRetries is the maximum number of reconnection attempts.
	// 0 means unlimited retries (until context cancellation or manual stop).
	MaxRetries int `yaml:"max_retries"`

	// MinBackoff is the initial backoff duration (e.g., "5s").
	MinBackoffStr string `yaml:"min_backoff,omitempty"`
	MinBackoff    time.Duration

	// MaxBackoff is the maximum backoff duration (e.g., "5m").
	MaxBackoffStr string `yaml:"max_backoff,omitempty"`
	MaxBackoff    time.Duration

	// BackoffFactor is the multiplier for exponential backoff (default: 2.0).
	BackoffFactor float64 `yaml:"backoff_factor"`
}

// SetDefaults sets default values for reconnection config.
func (rc *ReconnectionConfig) SetDefaults() {
	if rc.MinBackoff <= 0 {
		rc.MinBackoff = DefaultMinBackoff
	}
	if rc.MaxBackoff <= 0 {
		rc.MaxBackoff = DefaultMaxBackoff
	}
	if rc.MaxBackoff < rc.MinBackoff {
		rc.MaxBackoff = rc.MinBackoff * 2
	}
	if rc.BackoffFactor <= 1.0 {
		rc.BackoffFactor = DefaultBackoffFactor
	}
	if rc.MaxRetries < 0 {
		rc.MaxRetries = 0
	}
}

// Reconnector manages reconnection attempts with exponential backoff.
type Reconnector struct {
	logger *slog.Logger

	mu          sync.Mutex
	cfg         ReconnectionConfig
	attempts    int
	isRunning   bool
	lastBackoff time.Duration

	// Callback to attempt reconnection. Should return nil on success.
	reconnectFunc func(ctx context.Context) error
	// Callback to update server list before reconnection attempt (optional).
	refreshFunc func(ctx context.Context) error

	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// NewReconnector creates a new Reconnector with the given configuration.
func NewReconnector(
	logger *slog.Logger,
	cfg ReconnectionConfig,
	reconnectFunc func(ctx context.Context) error,
	refreshFunc func(ctx context.Context) error,
) *Reconnector {
	cfg.SetDefaults()

	return &Reconnector{
		logger:        logger,
		cfg:           cfg,
		reconnectFunc: reconnectFunc,
		refreshFunc:   refreshFunc,
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}
}

// Start begins the reconnection loop. Blocks until either:
// - a successful reconnection (returns nil)
// - max retries exceeded (returns error)
// - context cancelled (returns context error)
// - Stop() is called (returns ErrReconnectionStopped)
func (r *Reconnector) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("reconnector already running")
	}
	r.isRunning = true
	r.attempts = 0
	r.lastBackoff = 0
	r.stopCh = make(chan struct{})
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.isRunning = false
		close(r.stoppedCh)
		r.mu.Unlock()
	}()

	for {
		// Check if max retries reached
		r.mu.Lock()
		currentAttempt := r.attempts
		maxRetries := r.cfg.MaxRetries
		r.mu.Unlock()

		if maxRetries > 0 && currentAttempt >= maxRetries {
			return fmt.Errorf("reconnection failed after %d attempts (max retries exceeded)", currentAttempt)
		}

		// Calculate and wait for backoff
		backoff := r.nextBackoff()
		r.logger.Info("Waiting before reconnection attempt",
			"attempt", currentAttempt+1,
			"max_retries", formatMaxRetries(maxRetries),
			"backoff", backoff)

		if err := r.waitWithContext(ctx, backoff); err != nil {
			return err // context cancelled or stopped
		}

		// Attempt to refresh server list (if callback provided)
		if r.refreshFunc != nil {
			r.logger.Debug("Refreshing server list before reconnection attempt")
			if err := r.refreshFunc(ctx); err != nil {
				r.logger.Warn("Failed to refresh server list before reconnection", "error", err)
				// Continue anyway - might succeed with existing servers
			}
		}

		// Attempt reconnection
		r.logger.Info("Attempting reconnection",
			"attempt", currentAttempt+1,
			"max_retries", formatMaxRetries(maxRetries))

		err := r.reconnectFunc(ctx)
		if err == nil {
			r.logger.Info("Reconnection successful", "attempt", currentAttempt+1)
			r.mu.Lock()
			r.resetLocked()
			r.mu.Unlock()
			return nil
		}

		r.logger.Warn("Reconnection attempt failed",
			"attempt", currentAttempt+1,
			"error", err,
			"next_backoff", r.peekNextBackoff())

		r.mu.Lock()
		r.attempts++
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stopCh:
			return ErrReconnectionStopped
		default:
		}
	}
}

// Stop signals the reconnector to stop. Blocks until the loop exits.
func (r *Reconnector) Stop() {
	r.mu.Lock()
	select {
	case <-r.stopCh:
		// Already closed
	default:
		close(r.stopCh)
	}
	// Wait for the loop to finish (don't hold lock)
	waitCh := r.stoppedCh
	r.mu.Unlock()

	<-waitCh
}

// IsRunning returns whether the reconnector is currently active.
func (r *Reconnector) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRunning
}

// GetAttempts returns the current attempt count.
func (r *Reconnector) GetAttempts() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attempts
}

// GetStatus returns the current reconnection status.
func (r *Reconnector) GetStatus() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	remaining := "unlimited"
	if r.cfg.MaxRetries > 0 {
		remaining = fmt.Sprintf("%d", r.cfg.MaxRetries-r.attempts)
	}

	return map[string]interface{}{
		"is_running":     r.isRunning,
		"attempts":       r.attempts,
		"max_retries":    formatMaxRetries(r.cfg.MaxRetries),
		"remaining":      remaining,
		"last_backoff":   r.lastBackoff.String(),
		"min_backoff":    r.cfg.MinBackoff.String(),
		"max_backoff":    r.cfg.MaxBackoff.String(),
		"backoff_factor": r.cfg.BackoffFactor,
	}
}

// Reset resets the attempt counter and backoff state.
func (r *Reconnector) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetLocked()
}

// resetLocked resets state (must hold lock).
func (r *Reconnector) resetLocked() {
	r.attempts = 0
	r.lastBackoff = 0
}

// nextBackoff calculates the next backoff duration with exponential increase
// and jitter, capped at MaxBackoff.
func (r *Reconnector) nextBackoff() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts++

	// Cap at max retries check is done outside
	if r.lastBackoff == 0 {
		r.lastBackoff = r.cfg.MinBackoff
	} else {
		// Exponential increase: backoff = min(last * factor, max)
		next := time.Duration(float64(r.lastBackoff) * r.cfg.BackoffFactor)
		if next > r.cfg.MaxBackoff {
			next = r.cfg.MaxBackoff
		}
		r.lastBackoff = next
	}

	// Add jitter: +/- 25% for better distribution
	jitter := time.Duration(float64(r.lastBackoff) * 0.25 * (math.Sin(float64(r.attempts*7)) + 0.5))
	r.lastBackoff += jitter
	if r.lastBackoff > r.cfg.MaxBackoff {
		r.lastBackoff = r.cfg.MaxBackoff
	}

	return r.lastBackoff
}

// peekNextBackoff returns the next backoff without advancing the counter.
func (r *Reconnector) peekNextBackoff() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.lastBackoff == 0 {
		return r.cfg.MinBackoff
	}
	next := time.Duration(float64(r.lastBackoff) * r.cfg.BackoffFactor)
	if next > r.cfg.MaxBackoff {
		next = r.cfg.MaxBackoff
	}
	return next
}

// waitWithContext waits for the given duration, respecting context and stop channel.
func (r *Reconnector) waitWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.stopCh:
		return ErrReconnectionStopped
	case <-time.After(d):
		return nil
	}
}

// formatMaxRetries formats max retries for logging.
func formatMaxRetries(maxRetries int) string {
	if maxRetries <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", maxRetries)
}