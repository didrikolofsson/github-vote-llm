package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return &Logger{SugaredLogger: zap.New(core).Sugar()}, logs
}

func TestNew(t *testing.T) {
	log := New()
	if log == nil {
		t.Fatal("New() returned nil")
	}
	if log.SugaredLogger == nil {
		t.Fatal("SugaredLogger is nil")
	}
}

func TestNamed(t *testing.T) {
	log, logs := newTestLogger()
	child := log.Named("webhook")

	child.Infow("test message", "key", "value")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.LoggerName != "webhook" {
		t.Errorf("expected logger name %q, got %q", "webhook", entry.LoggerName)
	}
	if entry.Message != "test message" {
		t.Errorf("expected message %q, got %q", "test message", entry.Message)
	}

	found := false
	for _, f := range entry.ContextMap() {
		if f == "value" {
			found = true
		}
	}
	if !found {
		t.Error("expected structured field key=value in log entry")
	}
}

func TestNamedChaining(t *testing.T) {
	log, logs := newTestLogger()
	child := log.Named("server").Named("http")

	child.Infow("request handled")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	// zap separates nested names with "."
	if !strings.Contains(entries[0].LoggerName, "server") || !strings.Contains(entries[0].LoggerName, "http") {
		t.Errorf("expected logger name to contain server and http, got %q", entries[0].LoggerName)
	}
}

func TestLogLevels(t *testing.T) {
	log, logs := newTestLogger()

	log.Debugw("debug msg")
	log.Infow("info msg")
	log.Warnw("warn msg")
	log.Errorw("error msg")

	entries := logs.All()
	if len(entries) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(entries))
	}

	expected := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}

	for i, e := range entries {
		if e.Level != expected[i] {
			t.Errorf("entry %d: expected level %v, got %v", i, expected[i], e.Level)
		}
	}
}

func TestStructuredFields(t *testing.T) {
	log, logs := newTestLogger()

	log.Infow("issue processed",
		"issue", 42,
		"repo", "owner/repo",
		"action", "labeled",
	)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	ctx := entries[0].ContextMap()
	if ctx["issue"] != int64(42) {
		t.Errorf("expected issue=42, got %v", ctx["issue"])
	}
	if ctx["repo"] != "owner/repo" {
		t.Errorf("expected repo=owner/repo, got %v", ctx["repo"])
	}
	if ctx["action"] != "labeled" {
		t.Errorf("expected action=labeled, got %v", ctx["action"])
	}
}
