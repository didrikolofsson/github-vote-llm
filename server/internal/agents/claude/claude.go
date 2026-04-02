package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/didrikolofsson/github-vote-llm/internal/agents"
)

const BUFFER_SIZE = 1024 * 1024

type ClaudeRunner struct {
	apiKey  string
	workDir string
}

func NewClaudeRunner(apiKey, workDir string) *ClaudeRunner {
	return &ClaudeRunner{
		apiKey:  apiKey,
		workDir: workDir,
	}
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string) (<-chan agents.Event, error) {
	cmd := exec.CommandContext(ctx, "claude", "p", prompt)
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+r.apiKey)
	cmd.Dir = r.workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	ch := make(chan agents.Event, 16)
	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, BUFFER_SIZE), BUFFER_SIZE)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- agents.Event{Chunk: scanner.Text()}:
			}
		}

		if err := cmd.Wait(); err != nil {
			ch <- agents.Event{Err: fmt.Errorf("claude: %w", err)}
		}
	}()

	return ch, nil
}
