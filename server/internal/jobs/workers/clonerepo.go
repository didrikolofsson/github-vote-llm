package workers

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[args.CloneRepoArgs]
	svc    *services.GithubService
	runSvc *services.RunService
	jc     *river.Client[pgx.Tx]
}

func (w *CloneRepoWorker) Work(ctx context.Context, job *river.Job[args.CloneRepoArgs]) error {
	run, err := w.runSvc.GetRunByID(ctx, job.Args.RunID)
	if err != nil {
		return err
	}
	if run.Status == store.FeatureRunStatusCancelled {
		return nil
	}
	if err := w.svc.CloneRepoToWorkspace(ctx, job.Args.RunID); err != nil {
		return err
	}
	_, err = w.jc.Insert(ctx, args.RunAgentArgs{RunID: job.Args.RunID}, nil)
	return err
}
