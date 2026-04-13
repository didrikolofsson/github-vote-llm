package workers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[jobargs.CloneRepoArgs]
	db           *pgxpool.Pool
	GithubSvc    services.GithubService
	WorkspaceDir string
	RiverClient  *river.Client[pgx.Tx]
}

func NewCloneRepoWorker(
	db *pgxpool.Pool,
	githubSvc services.GithubService,
	workspaceDir string,
) (*CloneRepoWorker, error) {
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("creating workspace directory: %w", err)
	}
	return &CloneRepoWorker{
		db:           db,
		GithubSvc:    githubSvc,
		WorkspaceDir: workspaceDir,
	}, nil
}

var (
	ErrInvalidCloneURL        = errors.New("github: invalid or missing clone URL")
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

func (w *CloneRepoWorker) Work(ctx context.Context, job *river.Job[jobargs.CloneRepoArgs]) error {
	workspace := filepath.Join(w.WorkspaceDir, fmt.Sprint(job.Args.RunID))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}

	tx, err := w.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := store.New(tx)
	run, err := qtx.GetRunByID(ctx, job.Args.RunID)
	if err != nil {
		return err
	}

	if err := w.GithubSvc.CloneRepoToWorkspace(
		ctx, job.Args.UserID, job.Args.Owner, job.Args.Name, workspace,
	); err != nil {
		return err
	}

	if _, err := w.RiverClient.InsertTx(ctx, tx, jobargs.RunAgentArgs{
		Prompt:  run.Prompt,
		WorkDir: workspace,
		RunID:   job.Args.RunID,
	}, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
