package workers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[args.CloneRepoArgs]
	svc       *services.GithubService
	workspace string
}

var (
	ErrInvalidCloneURL        = errors.New("github: invalid or missing clone URL")
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

func (w *CloneRepoWorker) Work(ctx context.Context, job *river.Job[args.CloneRepoArgs]) error {
	workspace := filepath.Join(w.workspace, fmt.Sprint(job.Args.RunID))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}
	return w.svc.CloneRepoToWorkspace(ctx, job.Args.UserID, job.Args.Owner, job.Args.Name, workspace)
}
