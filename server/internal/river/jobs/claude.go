package jobs

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

// RunClaudeArgs must stay JSON-serializable: River persists args and reloads them
// in the worker process, so non-serializable types (*store.Queries, *oauth2.Config)
// must not be stored here—inject those on RunClaudeWorker instead.
type RunClaudeArgs struct {
	Prompt             string `json:"prompt"`
	UserID             int64  `json:"user_id"`
	TokenEncryptionKey string `json:"token_encryption_key"`
	Repository         *dtos.Repository `json:"repository"`
	Workspace          string           `json:"workspace"`
	ApiKey             string           `json:"api_key"`
}

func (RunClaudeArgs) Kind() string {
	return "run_claude"
}

type RunClaudeWorker struct {
	river.WorkerDefaults[RunClaudeArgs]
	Q              *store.Queries
	GithubOAuthCfg *oauth2.Config
}

func (w *RunClaudeWorker) Work(ctx context.Context, job *river.Job[RunClaudeArgs]) error {
	client := github.NewGithubClientByUserID(
		github.NewGithubClientByUserIDParams{
			Context:            ctx,
			Queries:            w.Q,
			Config:             w.GithubOAuthCfg,
			UserID:             job.Args.UserID,
			TokenEncryptionKey: job.Args.TokenEncryptionKey,
		},
	)

	// Clone target repo to workspace
	if err := os.MkdirAll(job.Args.Workspace, 0755); err != nil {
		return err
	}

	repo, _, err := client.Repositories.Get(
		ctx, job.Args.Repository.Owner, job.Args.Repository.Name,
	)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", *repo.CloneURL)
	cmd.Dir = job.Args.Workspace
	if err := cmd.Run(); err != nil {
		return err
	}

	// Run Claude in cloned repo
	fmt.Println("Running Claude in cloned repo")
	runner := claude.NewClaudeRunner(claude.NewClaudeRunnerParams{
		ApiKey:  job.Args.ApiKey,
		WorkDir: job.Args.Workspace + "/" + job.Args.Repository.Name,
	})
	ch, err := runner.Run(ctx, job.Args.Prompt)
	if err != nil {
		return err
	}

	for event := range ch {
		fmt.Println("Claude event: ", event.Chunk)
	}
	fmt.Println("Claude job completed")
	return nil
}
