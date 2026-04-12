package workers

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

type RunAgentWorker struct {
	river.WorkerDefaults[jobargs.RunAgentArgs]
	Queries           *store.Queries
	GithubOAuthConfig *oauth2.Config
}

func (w *RunAgentWorker) Work(ctx context.Context, job *river.Job[jobargs.RunAgentArgs]) error {
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
