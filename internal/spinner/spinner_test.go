package spinner

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinnerWritesOutput(t *testing.T) {
	var buf bytes.Buffer
	s := newWithWriter("loading...", &buf)
	s.Start()
	time.Sleep(250 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "loading...") {
		t.Errorf("expected output to contain %q, got %q", "loading...", out)
	}
}

func TestSpinnerContainsFrames(t *testing.T) {
	var buf bytes.Buffer
	s := newWithWriter("working", &buf)
	s.Start()
	time.Sleep(350 * time.Millisecond)
	s.Stop()

	out := buf.String()
	foundFrame := false
	for _, f := range frames {
		if strings.Contains(out, f) {
			foundFrame = true
			break
		}
	}
	if !foundFrame {
		t.Errorf("expected output to contain at least one spinner frame, got %q", out)
	}
}

func TestSpinnerUpdateMessage(t *testing.T) {
	var buf bytes.Buffer
	s := newWithWriter("first", &buf)
	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.UpdateMessage("second")
	time.Sleep(200 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "first") {
		t.Errorf("expected output to contain %q, got %q", "first", out)
	}
	if !strings.Contains(out, "second") {
		t.Errorf("expected output to contain %q, got %q", "second", out)
	}
}

func TestSpinnerStopIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	s := newWithWriter("test", &buf)
	s.Start()
	time.Sleep(150 * time.Millisecond)

	// Calling Stop multiple times should not panic
	s.Stop()
	s.Stop()
	s.Stop()
}

func TestSpinnerClearsLineOnStop(t *testing.T) {
	var buf bytes.Buffer
	s := newWithWriter("clearing", &buf)
	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	out := buf.String()
	// The last thing written should be the clear sequence \r\033[K
	if !strings.HasSuffix(out, "\r\033[K") {
		t.Errorf("expected output to end with line-clear escape, got trailing bytes %q", out[max(0, len(out)-20):])
	}
}

func TestNewUsesStderr(t *testing.T) {
	s := New("test")
	if s.out == nil {
		t.Fatal("expected non-nil writer")
	}
	s.Start()
	s.Stop()
}
