package main

import (
	"context"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	tests := []struct {
		current  time.Duration
		expected time.Duration
	}{
		{2 * time.Second, 4 * time.Second},
		{4 * time.Second, 8 * time.Second},
		{30 * time.Second, 60 * time.Second},
		{60 * time.Second, 60 * time.Second},  // capped at maxBackoff
		{120 * time.Second, 60 * time.Second}, // already over max
	}

	for _, tt := range tests {
		got := nextBackoff(tt.current)
		if got != tt.expected {
			t.Errorf("nextBackoff(%v) = %v, want %v", tt.current, got, tt.expected)
		}
	}
}

func TestWebhookForwarder_RestartsOnExit(t *testing.T) {
	var starts atomic.Int32

	f := &webhookForwarder{
		log: testLogger(),
	}
	f.newCommand = func() *exec.Cmd {
		starts.Add(1)
		// "true" exits immediately with code 0
		return exec.Command("true")
	}

	ctx, cancel := context.WithCancel(context.Background())

	f.cancel = cancel
	f.wg.Add(1)
	go f.supervise(ctx)

	// Wait until we see at least 2 starts (initial + at least one restart)
	deadline := time.After(10 * time.Second)
	for starts.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for restart, starts=%d", starts.Load())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	f.wg.Wait()

	if s := starts.Load(); s < 2 {
		t.Errorf("expected at least 2 starts, got %d", s)
	}
}

func TestWebhookForwarder_StopsCleanly(t *testing.T) {
	f := &webhookForwarder{
		log: testLogger(),
	}
	f.newCommand = func() *exec.Cmd {
		// "sleep 60" — a long-running process we'll cancel
		return exec.Command("sleep", "60")
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.wg.Add(1)
	go f.supervise(ctx)

	// Give time for the process to start
	time.Sleep(200 * time.Millisecond)

	// Cancel and verify it shuts down promptly
	cancel()

	done := make(chan struct{})
	go func() {
		f.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for clean shutdown")
	}
}

func TestWebhookForwarder_StopBeforeStart(t *testing.T) {
	f := &webhookForwarder{
		log: testLogger(),
	}
	// stop() should be safe to call even if start() was never called
	f.stop()
}

func TestWebhookForwarder_RestartsOnFailedStart(t *testing.T) {
	var attempts atomic.Int32

	f := &webhookForwarder{
		log: testLogger(),
	}
	f.newCommand = func() *exec.Cmd {
		n := attempts.Add(1)
		if n <= 2 {
			// First two attempts: command that doesn't exist
			return exec.Command("/nonexistent-binary-that-does-not-exist")
		}
		// Third attempt: succeeds but exits immediately
		return exec.Command("true")
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.wg.Add(1)
	go f.supervise(ctx)

	deadline := time.After(30 * time.Second)
	for attempts.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for attempts, got %d", attempts.Load())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	f.wg.Wait()

	if a := attempts.Load(); a < 3 {
		t.Errorf("expected at least 3 attempts, got %d", a)
	}
}
