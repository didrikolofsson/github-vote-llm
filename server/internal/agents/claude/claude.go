package claude

import (
	"context"
	"os"
	"os/exec"
)

const BUFFER_SIZE = 1024 * 1024

type ClaudeRunner struct {
	apiKey string
}

type NewClaudeRunnerParams struct {
	ApiKey string
}

func NewClaudeRunner(p NewClaudeRunnerParams) *ClaudeRunner {
	return &ClaudeRunner{
		apiKey: p.ApiKey,
	}
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string, workDir string) error {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+r.apiKey)
	cmd.Dir = workDir
	return cmd.Run()
}
