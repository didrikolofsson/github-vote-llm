package workers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type OpenPRWorker struct {
	river.WorkerDefaults[jobargs.OpenPRArgs]
	db           *pgxpool.Pool
	GithubSvc    services.GithubService
	WorkspaceDir string
}

func NewOpenPRWorker(db *pgxpool.Pool, githubSvc services.GithubService, workspaceDir string) *OpenPRWorker {
	return &OpenPRWorker{db: db, GithubSvc: githubSvc, WorkspaceDir: workspaceDir}
}

func (w *OpenPRWorker) Work(ctx context.Context, job *river.Job[jobargs.OpenPRArgs]) error {
	title := prTitle(job.Args.Prompt)
	_, err := w.GithubSvc.OpenPR(
		ctx,
		job.Args.UserID,
		job.Args.Owner,
		job.Args.Name,
		job.Args.BranchName,
		title,
		job.Args.Prompt,
	)
	if err != nil {
		return fmt.Errorf("opening PR: %w", err)
	}

	q := store.New(w.db)
	if err := q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
		Status: store.FeatureRunStatusCompleted,
		ID:     job.Args.RunID,
	}); err != nil {
		return err
	}

	workspace := filepath.Join(w.WorkspaceDir, fmt.Sprint(job.Args.RunID))
	if err := os.RemoveAll(workspace); err != nil {
		return fmt.Errorf("cleaning up workspace: %w", err)
	}

	return nil
}

// prTitle derives a short PR title from the prompt.
func prTitle(prompt string) string {
	const maxLen = 72
	if len(prompt) <= maxLen {
		return prompt
	}
	return prompt[:maxLen-3] + "..."
}
