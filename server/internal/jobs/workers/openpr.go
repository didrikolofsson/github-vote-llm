package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type OpenPRWorker struct {
	river.WorkerDefaults[args.OpenRepoPullRequestArgs]
	svc *services.GithubService
	log *logger.Logger
}

// func (w *OpenPRWorker) Work(ctx context.Context, job *river.Job[args.OpenRepoPullRequestArgs]) error {
// 	a := job.Args

// 	if err := w.svc.PushBranch(ctx, a.OrganizationID, a.WorktreeDir, a.Owner, a.Name, a.BranchName); err != nil {
// 		return fmt.Errorf("push branch: %w", err)
// 	}

// 	title := a.Prompt
// 	if len(title) > 72 {
// 		title = title[:72]
// 	}
// 	body := fmt.Sprintf("Automated PR opened by the vote-llm agent.\n\n**Prompt:**\n%s", a.Prompt)

// 	prURL, err := w.svc.OpenPR(ctx, a.OrganizationID, a.Owner, a.Name, a.BranchName, title, body)
// 	if err != nil {
// 		return fmt.Errorf("open PR: %w", err)
// 	}

// 	w.log.Infow("PR opened", "url", prURL, "run_id", a.RunID, "branch", a.BranchName)
// 	return nil
// }
