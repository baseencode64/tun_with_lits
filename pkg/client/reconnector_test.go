package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// testLogger is a logger that discards output during tests
var testLoggerReconn = slog.New(slog.NewTextHandler(nil, nil))

func TestReconnector_NextBackoff(t *testing.T) {
	t.Run("first backoff is min_backoff", func(t *testing.T) {
		r := &Reconnector{
			logger: testLoggerReconn,
			cfg: ReconnectionConfig{
				MinBackoff:    5 * time.Second,
				MaxBackoff:    5 * time.Minute,
				BackoffFactor: 2.0,
			},
			stopCh:    make(chan struct{}),
			stoppedCh: make(chan struct{}),
		}

		b1 := r.nextBackoff()
		// First backoff should be close to min (with jitter)
		if b1 < r.cfg.MinBackoff-2*time.Second || b1 > r.cfg.MinBackoff+2*time.Second {
			t.Errorf("first backoff %v should be near min_backoff %v", b1, r.cfg.MinBackoff)
		}
	})

	t.Run("backoff increases exponentially", func(t *testing.T) {
		r := &Reconnector{
			logger: testLoggerReconn,
			cfg: ReconnectionConfig{
				MinBackoff:    1 * time.Second,
				MaxBackoff:    1 * time.Minute,
				BackoffFactor: 2.0,
			},
			stopCh:    make(chan struct{}),
			stoppedCh: make(chan struct{}),
		}

		b1 := r.nextBackoff()
		b2 := r.nextBackoff()
		b3 := r.nextBackoff()

		t.Logf("Backoffs: b1=%v, b2=%v, b3=%v", b1, b2, b3)

		// Each should be >= previous (exponential growth with jitter)
		if b2 < b1 || b3 < b2 {
			t.Errorf("backoff should increase: b1=%v, b2=%v, b3=%v", b1, b2, b3)
		}
	})

	t.Run("backoff is capped at max", func(t *testing.T) {
		r := &Reconnector{
			logger: testLoggerReconn,
			cfg: ReconnectionConfig{
				MinBackoff:    1 * time.Second,
				MaxBackoff:    3 * time.Second,
				BackoffFactor: 10.0, // Aggressive growth
			},
			stopCh:    make(chan struct{}),
			stoppedCh: make(chan struct{}),
		}

		// Even with huge factor, should not exceed max
		for i := 0; i < 10; i++ {
			b := r.nextBackoff()
			if b > r.cfg.MaxBackoff+time.Second {
				t.Errorf("backoff %v exceeded max %v", b, r.cfg.MaxBackoff)
			}
		}
	})

	t.Run("reset resets backoff", func(t *testing.T) {
		r := &Reconnector{
			logger: testLoggerReconn,
			cfg: ReconnectionConfig{
				MinBackoff:    1 * time.Second,
				MaxBackoff:    1 * time.Minute,
				BackoffFactor: 2.0,
			},
			stopCh:    make(chan struct{}),
			stoppedCh: make(chan struct{}),
		}

		b1 := r.nextBackoff()
		r.Reset()
		b2 := r.nextBackoff()

		if b2 < r.cfg.MinBackoff-time.Second || b2 > r.cfg.MinBackoff+time.Second {
			t.Errorf("after reset, backoff %v should be near min_backoff %v", b2, r.cfg.MinBackoff)
		}
		_ = b1 // b1 is used but not needed for assertion
	})
}

func TestReconnector_Start_SuccessOnFirstAttempt(t *testing.T) {
	attemptCount := 0
	reconnectFunc := func(ctx context.Context) error {
		attemptCount++
		return nil // Success
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    100 * time.Millisecond,
			MaxBackoff:    1 * time.Second,
			BackoffFactor: 2.0,
			MaxRetries:    3,
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.Start(ctx)
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}

	if attemptCount != 0 {
		t.Errorf("expected 0 attempts (success on first), got %d", attemptCount)
	}
}

func TestReconnector_Start_RetriesAndSucceeds(t *testing.T) {
	var attemptCount int32
	reconnectFunc := func(ctx context.Context) error {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			return fmt.Errorf("attempt %d failed", count)
		}
		return nil // Success on 3rd attempt
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    50 * time.Millisecond,
			MaxBackoff:    500 * time.Millisecond,
			BackoffFactor: 2.0,
			MaxRetries:    5,
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.Start(ctx)
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}

	attempts := atomic.LoadInt32(&attemptCount)
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	t.Logf("Reconnection succeeded after %d attempts", attempts)
}

func TestReconnector_Start_MaxRetriesExceeded(t *testing.T) {
	reconnectFunc := func(ctx context.Context) error {
		return errors.New("permanent failure")
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    10 * time.Millisecond,
			MaxBackoff:    100 * time.Millisecond,
			BackoffFactor: 2.0,
			MaxRetries:    3, // Only 3 retries
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.Start(ctx)
	if err == nil {
		t.Fatal("expected error (max retries exceeded), got nil")
	}

	if !contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected max retries exceeded error, got: %v", err)
	}

	t.Logf("Got expected error: %v", err)
}

func TestReconnector_Start_ContextCancelled(t *testing.T) {
	reconnectFunc := func(ctx context.Context) error {
		return errors.New("still failing")
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    100 * time.Millisecond,
			MaxBackoff:    1 * time.Second,
			BackoffFactor: 2.0,
			MaxRetries:    0, // Unlimited
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := r.Start(ctx)
	if err == nil {
		t.Fatal("expected context cancelled error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}

	t.Logf("Got expected error: %v", err)
}

func TestReconnector_Start_Stop(t *testing.T) {
	reconnectFunc := func(ctx context.Context) error {
		return errors.New("still failing")
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    100 * time.Millisecond,
			MaxBackoff:    1 * time.Second,
			BackoffFactor: 2.0,
			MaxRetries:    0, // Unlimited
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stop after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		r.Stop()
	}()

	err := r.Start(ctx)
	if err != ErrReconnectionStopped {
		t.Errorf("expected ErrReconnectionStopped, got: %v", err)
	}

	t.Logf("Got expected error: %v", err)
}

func TestReconnector_RefreshFuncCalled(t *testing.T) {
	var refreshCount int32
	reconnectFunc := func(ctx context.Context) error {
		return errors.New("still failing")
	}
	refreshFunc := func(ctx context.Context) error {
		atomic.AddInt32(&refreshCount, 1)
		return nil
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    10 * time.Millisecond,
			MaxBackoff:    100 * time.Millisecond,
			BackoffFactor: 2.0,
			MaxRetries:    2, // Only 2 attempts
		},
		reconnectFunc,
		refreshFunc,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail after max retries
	_ = r.Start(ctx) // Ignore error, we expect failure

	refreshes := atomic.LoadInt32(&refreshCount)
	if refreshes < 1 {
		t.Errorf("expected at least 1 refresh call, got %d", refreshes)
	}

	t.Logf("Refresh function called %d times", refreshes)
}

func TestReconnector_GetStatus(t *testing.T) {
	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    5 * time.Second,
			MaxBackoff:    5 * time.Minute,
			BackoffFactor: 2.0,
			MaxRetries:    5,
		},
		func(ctx context.Context) error { return nil },
		nil,
	)

	status := r.GetStatus()

	if status["is_running"] != false {
		t.Errorf("expected is_running=false, got %v", status["is_running"])
	}
	if status["max_retries"] != "5" {
		t.Errorf("expected max_retries=5, got %v", status["max_retries"])
	}
	if status["backoff_factor"] != 2.0 {
		t.Errorf("expected backoff_factor=2.0, got %v", status["backoff_factor"])
	}
}

func TestReconnector_AlreadyRunning(t *testing.T) {
	reconnectFunc := func(ctx context.Context) error {
		time.Sleep(5 * time.Second)
		return nil
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    100 * time.Millisecond,
			MaxBackoff:    1 * time.Second,
			BackoffFactor: 2.0,
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Don't start, just check that starting twice gives error
	// We need to manually set isRunning
	r.mu.Lock()
	r.isRunning = true
	r.mu.Unlock()

	err := r.Start(ctx)
	if err == nil {
		t.Error("expected error for already running, got nil")
	}
	if err != nil && !contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' error, got: %v", err)
	}
}

func TestReconnector_UnlimitedRetries(t *testing.T) {
	var attemptCount int32
	reconnectFunc := func(ctx context.Context) error {
		count := atomic.AddInt32(&attemptCount, 1)
		// Succeed after 5 attempts
		if count >= 5 {
			return nil
		}
		return errors.New("still failing")
	}

	r := NewReconnector(
		testLoggerReconn,
		ReconnectionConfig{
			MinBackoff:    10 * time.Millisecond,
			MaxBackoff:    100 * time.Millisecond,
			BackoffFactor: 1.5,
			MaxRetries:    0, // 0 = unlimited
		},
		reconnectFunc,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.Start(ctx)
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}

	attempts := atomic.LoadInt32(&attemptCount)
	if attempts != 5 {
		t.Errorf("expected 5 attempts, got %d", attempts)
	}

	t.Logf("Unlimited retries: succeeded after %d attempts", attempts)
}

func TestReconnector_SetDefaults(t *testing.T) {
	cfg := ReconnectionConfig{}
	cfg.SetDefaults()

	if cfg.MinBackoff != DefaultMinBackoff {
		t.Errorf("expected MinBackoff=%v, got %v", DefaultMinBackoff, cfg.MinBackoff)
	}
	if cfg.MaxBackoff != DefaultMaxBackoff {
		t.Errorf("expected MaxBackoff=%v, got %v", DefaultMaxBackoff, cfg.MaxBackoff)
	}
	if cfg.BackoffFactor != DefaultBackoffFactor {
		t.Errorf("expected BackoffFactor=%v, got %v", DefaultBackoffFactor, cfg.BackoffFactor)
	}
	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected MaxRetries=%v, got %v", DefaultMaxRetries, cfg.MaxRetries)
	}
}

func TestReconnector_SetDefaultsPreservesExisting(t *testing.T) {
	cfg := ReconnectionConfig{
		MinBackoff:    10 * time.Second,
		MaxBackoff:    30 * time.Second,
		BackoffFactor: 3.0,
		MaxRetries:    10,
	}
	cfg.SetDefaults()

	if cfg.MinBackoff != 10*time.Second {
		t.Errorf("expected MinBackoff=10s, got %v", cfg.MinBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("expected MaxBackoff=30s, got %v", cfg.MaxBackoff)
	}
	if cfg.BackoffFactor != 3.0 {
		t.Errorf("expected BackoffFactor=3.0, got %v", cfg.BackoffFactor)
	}
	if cfg.MaxRetries != 10 {
		t.Errorf("expected MaxRetries=10, got %v", cfg.MaxRetries)
	}
}

func TestReconnector_Reset(t *testing.T) {
	r := &Reconnector{
		logger: testLoggerReconn,
		cfg: ReconnectionConfig{
			MinBackoff:    1 * time.Second,
			MaxBackoff:    1 * time.Minute,
			BackoffFactor: 2.0,
		},
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}

	// Advance state
	b1 := r.nextBackoff()
	b2 := r.nextBackoff()

	// Reset
	r.Reset()

	if r.attempts != 0 {
		t.Errorf("expected attempts=0 after reset, got %d", r.attempts)
	}

	// After reset, next backoff should be near min again
	b3 := r.nextBackoff()
	if b3 < r.cfg.MinBackoff-time.Second || b3 > r.cfg.MinBackoff+time.Second {
		t.Errorf("after reset, backoff %v should be near min_backoff %v", b3, r.cfg.MinBackoff)
	}

	_ = b1
	_ = b2
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}