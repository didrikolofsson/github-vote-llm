package workers

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type RunAgentWorker struct {
	river.WorkerDefaults[jobargs.RunAgentArgs]
	ApiKey      string
	RiverClient *river.Client[pgx.Tx]
}

func NewRunAgentWorker(apiKey string) *RunAgentWorker {
	return &RunAgentWorker{ApiKey: apiKey}
}

func (w *RunAgentWorker) Work(ctx context.Context, job *river.Job[jobargs.RunAgentArgs]) error {
	fmt.Println("Running Claude in cloned repo")
	runner := claude.NewClaudeRunner(claude.NewClaudeRunnerParams{
		ApiKey:  w.ApiKey,
		WorkDir: job.Args.WorkDir,
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
