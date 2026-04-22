package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type CreateRunParams struct {
	Prompt    string
	FeatureID int64
	UserID    int64
}

type RunService struct {
	db     *pgxpool.Pool
	q      *store.Queries
	jc     *river.Client[pgx.Tx]
	env    *config.Environment
	runner *claude.ClaudeRunner
}

func NewRunService(db *pgxpool.Pool, q *store.Queries, env *config.Environment, jc *river.Client[pgx.Tx], runner *claude.ClaudeRunner) *RunService {
	return &RunService{db: db, q: q, env: env, jc: jc, runner: runner}
}

func storeToRunDTO(run store.FeatureRun) *dtos.RunDTO {
	return &dtos.RunDTO{
		ID:        run.ID,
		Prompt:    run.Prompt,
		FeatureID: run.FeatureID,
		Status:    dtos.RunStatus(run.Status),
	}
}

func CreateSandboxDir(workspace string, organizationID int64, repositoryID int64) (string, error) {
	workspaceTrimmed := strings.TrimSuffix(workspace, "/")
	dir := filepath.Join(workspaceTrimmed, fmt.Sprintf("%d/%d", organizationID, repositoryID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *RunService) CreateRun(ctx context.Context, p CreateRunParams) (*dtos.RunDTO, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	repo, err := qtx.GetRepositoryByFeatureID(ctx, p.FeatureID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	workspace, err := CreateSandboxDir(s.env.WORKSPACE_DIR, repo.OrganizationID, repo.ID)
	if err != nil {
		return nil, err
	}

	run, err := qtx.CreateRun(ctx, store.CreateRunParams{
		Prompt:          p.Prompt,
		FeatureID:       p.FeatureID,
		Status:          store.FeatureRunStatusPending,
		CreatedByUserID: p.UserID,
		Workspace:       workspace,
	})
	if err != nil {
		if api_errors.IsForeignKeyViolationErr(err) {
			return nil, ErrFeatureNotFound
		}
		return nil, err
	}

	if _, err := s.jc.InsertTx(ctx, tx, args.CloneRepoArgs{
		UserID: p.UserID,
		RunID:  run.ID,
	}, nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return storeToRunDTO(run), nil
}

func prepareWorktree(ctx context.Context, repoDir, worktreeDir, branch string) error {
	fetch := exec.CommandContext(ctx, "git", "-C", repoDir, "fetch", "origin", "main")
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch: %w: %s", err, out)
	}

	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return err
	}

	add := exec.CommandContext(ctx, "git", "-C", repoDir, "worktree", "add", "-b", branch, worktreeDir, "origin/main")
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return nil
}

func (s *RunService) RunAgent(ctx context.Context, userID, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		return err
	}

	if err := s.q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
		Status: store.FeatureRunStatusRunning,
		ID:     runID,
	}); err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	repoDir := filepath.Join(run.Workspace, run.RepositoryName)
	worktreeDir := filepath.Join(run.Workspace, "worktrees", fmt.Sprintf("run-%d", run.ID))
	branch := fmt.Sprintf("feature-%d-run-%d", run.FeatureID, run.ID)

	if err := prepareWorktree(ctx, repoDir, worktreeDir, branch); err != nil {
		if err := s.q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
			Status: store.FeatureRunStatusFailed,
			ID:     runID,
		}); err != nil {
			return fmt.Errorf("failed to update run status: %w", err)
		}
		return fmt.Errorf("failed to prepare worktree: %w", err)
	}

	if err := s.runner.Run(ctx, run.Prompt, worktreeDir); err != nil {
		if err := s.q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
			Status: store.FeatureRunStatusFailed,
			ID:     runID,
		}); err != nil {
			return fmt.Errorf("failed to update run status: %w", err)
		}
		return fmt.Errorf("failed to run agent: %w", err)
	}

	if err := s.q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
		Status: store.FeatureRunStatusCompleted,
		ID:     runID,
	}); err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	return nil
}
