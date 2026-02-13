package spinner

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// frames defines the spinner animation characters.
var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated spinner with a message on a terminal.
type Spinner struct {
	mu      sync.Mutex
	message string
	out     io.Writer
	stop    chan struct{}
	done    chan struct{}
}

// New creates a new Spinner that writes to stderr.
func New(message string) *Spinner {
	return &Spinner{
		message: message,
		out:     os.Stderr,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// newWithWriter creates a Spinner with a custom writer (for testing).
func newWithWriter(message string, w io.Writer) *Spinner {
	return &Spinner{
		message: message,
		out:     w,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation in a background goroutine.
func (s *Spinner) Start() {
	go s.run()
}

func (s *Spinner) run() {
	defer close(s.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-s.stop:
			// Clear the spinner line
			fmt.Fprintf(s.out, "\r\033[K")
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()
			fmt.Fprintf(s.out, "\r\033[K%s %s", frames[i%len(frames)], msg)
			i++
		}
	}
}

// UpdateMessage changes the spinner's displayed message.
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop halts the spinner animation and clears the line.
func (s *Spinner) Stop() {
	select {
	case <-s.stop:
		// Already stopped
		return
	default:
		close(s.stop)
	}
	<-s.done
}
