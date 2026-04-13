package workers

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[jobargs.CloneRepoArgs]
	db          *pgxpool.Pool
	Services    *services.Services
	RiverClient *river.Client[pgx.Tx]
}

var (
	ErrInvalidCloneURL        = errors.New("github: invalid or missing clone URL")
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

func (w *CloneRepoWorker) Work(
	ctx context.Context, job *river.Job[jobargs.CloneRepoArgs],
) error {
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
	env, err := config.LoadEnv()
	if err != nil {
		return err
	}
	if err := w.Services.GithubService.CloneRepoToWorkspace(
		ctx, job.Args.UserID, job.Args.Owner, job.Args.Name, job.Args.Workspace,
	); err != nil {
		return err
	}

	w.RiverClient.InsertTx(ctx, tx, &jobargs.RunAgentArgs{
		Prompt:  run.Prompt,
		WorkDir: job.Args.Workspace,
		ApiKey:  env.ANTHROPIC_API_KEY,
	}, nil)

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
