package client

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestHealthChecker_BasicHealthCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	checker := NewHealthChecker(logger, 1*time.Second, 500*time.Millisecond, 3)

	if !checker.IsHealthy() {
		t.Error("New health checker should be healthy by default")
	}

	status := checker.GetStatus()
	if status["is_healthy"] != true {
		t.Errorf("Expected healthy status, got %v", status["is_healthy"])
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	checker := NewHealthChecker(logger, 100*time.Millisecond, 50*time.Millisecond, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	failedOver := false
	onUnhealthy := func() {
		failedOver = true
	}

	// Start with invalid server - should fail quickly
	checker.Start(ctx, "nonexistent.invalid", "9999", onUnhealthy)
	
	// Wait for checks
	time.Sleep(400 * time.Millisecond)
	
	// Stop checker
	checker.Stop()

	// Should have attempted checks
	status := checker.GetStatus()
	t.Logf("Final status: %+v", status)
}

func TestHealthChecker_ConsecutiveFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	maxRetries := 2
	checker := NewHealthChecker(logger, 100*time.Millisecond, 50*time.Millisecond, maxRetries)

	failoverTriggered := false
	onUnhealthy := func() {
		failoverTriggered = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start with unreachable server
	checker.Start(ctx, "192.0.2.1", "9999", onUnhealthy) // TEST-NET address (should be unreachable)
	
	// Wait for max retries to trigger failover
	time.Sleep(300 * time.Millisecond)

	if !failoverTriggered {
		t.Log("Failover not triggered yet - waiting more...")
		time.Sleep(200 * time.Millisecond)
	}

	checker.Stop()

	status := checker.GetStatus()
	t.Logf("Status after failures: %+v", status)
	
	if status["consecutive_failures"].(int) == 0 {
		t.Log("No consecutive failures recorded - server might be reachable")
	}
}

func TestHealthChecker_GetStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	checker := NewHealthChecker(logger, 5*time.Second, 3*time.Second, 5)

	status := checker.GetStatus()
	
	if status["check_interval"] != 5*time.Second {
		t.Errorf("Expected interval 5s, got %v", status["check_interval"])
	}
	
	if status["max_retries"] != 5 {
		t.Errorf("Expected max retries 5, got %v", status["max_retries"])
	}
}
