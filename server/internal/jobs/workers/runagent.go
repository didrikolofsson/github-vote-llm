package workers

import (
	"context"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type RunAgentWorker struct {
	river.WorkerDefaults[args.RunAgentArgs]
	svc *services.RunService
	jc  *river.Client[pgx.Tx]
}

func (w *RunAgentWorker) Timeout(*river.Job[args.RunAgentArgs]) time.Duration {
	return 30 * time.Minute
}

func (w *RunAgentWorker) Work(ctx context.Context, job *river.Job[args.RunAgentArgs]) error {
	return w.svc.RunAgent(ctx, job.Args.UserID, job.Args.RunID)
}
