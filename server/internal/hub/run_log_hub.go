package hub

import (
	"sync"
	"time"
)

// RunLogHub fans out per-run agent output lines to SSE subscribers.
// Lines are also buffered in memory so late subscribers get the history.
type RunLogHub struct {
	mu   sync.Mutex
	runs map[int64]*runLogEntry
}

type runLogEntry struct {
	lines []string
	subs  []chan string
	done  bool
}

func NewRunLogHub() *RunLogHub {
	return &RunLogHub{runs: make(map[int64]*runLogEntry)}
}

func (h *RunLogHub) entry(runID int64) *runLogEntry {
	e, ok := h.runs[runID]
	if !ok {
		e = &runLogEntry{}
		h.runs[runID] = e
	}
	return e
}

// Publish sends a line to all active subscribers and appends it to the buffer.
func (h *RunLogHub) Publish(runID int64, line string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e := h.entry(runID)
	if e.done {
		return
	}
	e.lines = append(e.lines, line)
	for _, ch := range e.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

// Subscribe returns a snapshot of existing lines and a channel for future ones.
// The channel is closed when the run finishes. If the run is already done,
// existing lines are returned and ch is nil.
func (h *RunLogHub) Subscribe(runID int64) (existing []string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e := h.entry(runID)
	existing = make([]string, len(e.lines))
	copy(existing, e.lines)
	if e.done {
		return existing, nil
	}
	ch = make(chan string, 2000)
	e.subs = append(e.subs, ch)
	return existing, ch
}

// Unsubscribe removes a channel from the subscriber list.
func (h *RunLogHub) Unsubscribe(runID int64, ch chan string) {
	if ch == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.runs[runID]
	if !ok {
		return
	}
	for i, s := range e.subs {
		if s == ch {
			e.subs = append(e.subs[:i], e.subs[i+1:]...)
			return
		}
	}
}

// Close marks the run as finished, closing all subscriber channels.
// Memory is released after a short grace period for late readers.
func (h *RunLogHub) Close(runID int64) {
	h.mu.Lock()
	e, ok := h.runs[runID]
	if !ok || e.done {
		h.mu.Unlock()
		return
	}
	e.done = true
	for _, ch := range e.subs {
		close(ch)
	}
	e.subs = nil
	h.mu.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.runs, runID)
	}()
}
