package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

const (
	initialBackoff = 2 * time.Second
	maxBackoff     = 60 * time.Second
	backoffFactor  = 2
)

// webhookForwarder supervises the `gh webhook forward` subprocess,
// automatically restarting it on unexpected exits (e.g. websocket close 1006).
type webhookForwarder struct {
	client *gh.Client
	owner  string
	repo   string
	port   int
	secret string
	log    *logger.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup

	// newCommand builds the exec.Cmd to run. Overridable for testing.
	newCommand func() *exec.Cmd
}

func newWebhookForwarder(client *gh.Client, owner, repo string, port int, secret string, log *logger.Logger) *webhookForwarder {
	f := &webhookForwarder{
		client: client,
		owner:  owner,
		repo:   repo,
		port:   port,
		secret: secret,
		log:    log.Named("dev"),
	}
	f.newCommand = f.defaultNewCommand
	return f
}

func (f *webhookForwarder) defaultNewCommand() *exec.Cmd {
	args := []string{
		"webhook", "forward",
		"--repo=" + fmt.Sprintf("%s/%s", f.owner, f.repo),
		"--events=issues,issue_comment",
		fmt.Sprintf("--url=http://localhost:%d/webhook", f.port),
		"--secret=" + f.secret,
	}
	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// start begins the supervised forwarding loop. It cleans up existing webhooks,
// starts the subprocess, and monitors it in a goroutine that restarts on failure.
func (f *webhookForwarder) start(ctx context.Context) error {
	if err := f.client.RemoveLocalRepoWebhooks(ctx, f.owner, f.repo, f.port); err != nil {
		return fmt.Errorf("remove local repo webhooks: %w", err)
	}

	loopCtx, cancel := context.WithCancel(ctx)
	f.cancel = cancel

	f.wg.Add(1)
	go f.supervise(loopCtx)

	return nil
}

// stop signals the supervisor to stop and waits for cleanup.
func (f *webhookForwarder) stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()
}

// supervise runs the gh webhook forward process in a loop, restarting on
// unexpected exits with exponential backoff.
func (f *webhookForwarder) supervise(ctx context.Context) {
	defer f.wg.Done()

	backoff := initialBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		cmd := f.newCommand()
		if err := cmd.Start(); err != nil {
			f.log.Errorw("failed to start gh webhook forward", "error", err)
			if !f.backoffWait(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		f.log.Infow("started gh webhook forward", "repo", f.owner+"/"+f.repo)

		// Wait for process exit in a goroutine so we can also respond to ctx cancellation.
		exitCh := make(chan error, 1)
		go func() {
			exitCh <- cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			// Shutting down — kill the process and return.
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			<-exitCh
			f.log.Infow("stopped gh webhook forward")
			return

		case err := <-exitCh:
			// Process exited unexpectedly — restart after backoff.
			f.log.Warnw("gh webhook forward exited, will restart",
				"error", err,
				"backoff", backoff.String(),
			)
			if !f.backoffWait(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
		}
	}
}

// backoffWait sleeps for the given duration, returning false if the context is cancelled.
func (f *webhookForwarder) backoffWait(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func nextBackoff(current time.Duration) time.Duration {
	return min(current*backoffFactor, maxBackoff)
}
