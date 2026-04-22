package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type OpenPRWorker struct {
	river.WorkerDefaults[args.OpenRepoPullRequestArgs]
	svc services.GithubService
}

// func (w *OpenPRWorker) Work(ctx context.Context, job *river.Job[args.OpenRepoPullRequestArgs]) error {
// 	return w.svc.OpenRepoPullRequest(ctx, job.Args.UserID, job.Args.Owner, job.Args.Name, job.Args.BranchName, job.Args.Prompt)
// }
