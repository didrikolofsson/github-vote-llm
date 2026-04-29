package workers

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[args.CloneRepoArgs]
	svc *services.GithubService
}

func (w *CloneRepoWorker) Work(ctx context.Context, job *river.Job[args.CloneRepoArgs]) error {
	return w.svc.CloneRepoToWorkspace(ctx, job.Args.RunID)
}
