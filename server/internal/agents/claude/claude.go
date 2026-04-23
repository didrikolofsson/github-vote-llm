package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

const BUFFER_SIZE = 1024 * 1024

type ClaudeRunner struct {
	apiKey string
	log    *logger.Logger
}

type NewClaudeRunnerParams struct {
	ApiKey string
	Logger *logger.Logger
}

func NewClaudeRunner(p NewClaudeRunnerParams) *ClaudeRunner {
	return &ClaudeRunner{
		apiKey: p.ApiKey,
		log:    p.Logger.Named("claude"),
	}
}

// runLog is a mutex-protected file writer shared across concurrent stream goroutines.
type runLog struct {
	mu   sync.Mutex
	file *os.File
}

func (rl *runLog) writeLine(stream, line string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	fmt.Fprintf(rl.file, "[%s] %s\n", stream, line)
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string, workDir string) error {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--verbose", "--dangerously-skip-permissions")
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+r.apiKey)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Tee output to a log file alongside the worktree so it can be tailed:
	//   tail -f <workDir>.log
	logPath := workDir + ".log"
	var rl *runLog
	if f, err := os.Create(logPath); err != nil {
		r.log.Warnw("failed to create run log file", "path", logPath, "err", err)
	} else {
		rl = &runLog{file: f}
		defer f.Close()
		r.log.Infow("claude start", "dir", workDir, "log", logPath, "prompt_chars", len(prompt))
	}

	if rl == nil {
		r.log.Infow("claude start", "dir", workDir, "prompt_chars", len(prompt))
	}

	start := time.Now()

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go r.streamLines(&wg, stdout, "stdout", rl)
	go r.streamLines(&wg, stderr, "stderr", rl)
	wg.Wait()

	err = cmd.Wait()
	r.log.Infow("claude exit", "dir", workDir, "duration", time.Since(start).String(), "err", err)
	return err
}

func (r *ClaudeRunner) streamLines(wg *sync.WaitGroup, rc io.ReadCloser, stream string, rl *runLog) {
	defer wg.Done()
	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 64*1024), BUFFER_SIZE)
	for scanner.Scan() {
		line := scanner.Text()
		r.log.Infow(line, "stream", stream)
		if rl != nil {
			rl.writeLine(stream, line)
		}
	}
	if err := scanner.Err(); err != nil {
		r.log.Warnw("stream scan error", "stream", stream, "err", err)
	}
}
