package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

var (
	ErrRunNotFound       = errors.New("run not found")
	ErrRunNotCancellable = errors.New("run is not in a cancellable state")
	ErrRunNotDeletable   = errors.New("only cancelled runs can be deleted")
)

type CreateRunParams struct {
	Prompt    string
	FeatureID int64
	UserID    int64
}

type RunService struct {
	db        *pgxpool.Pool
	q         *store.Queries
	jc        *river.Client[pgx.Tx]
	env       *config.Environment
	runner    *claude.ClaudeRunner
	hub       hub.Hub
	githubSvc *GithubService
}

func NewRunService(db *pgxpool.Pool, q *store.Queries, env *config.Environment, jc *river.Client[pgx.Tx], runner *claude.ClaudeRunner, h hub.Hub, githubSvc *GithubService) *RunService {
	return &RunService{db: db, q: q, env: env, jc: jc, runner: runner, hub: h, githubSvc: githubSvc}
}

func storeToRunDTO(run store.FeatureRun) *dtos.RunDTO {
	var completedAt *time.Time
	if run.CompletedAt.Valid {
		completedAt = &run.CompletedAt.Time
	}

	return &dtos.RunDTO{
		ID:              run.ID,
		Prompt:          run.Prompt,
		FeatureID:       run.FeatureID,
		Status:          dtos.RunStatus(run.Status),
		CreatedByUserID: run.CreatedByUserID,
		CreatedAt:       run.CreatedAt.Time,
		CompletedAt:     completedAt,
		PRURL:           run.PrUrl,
	}
}

func listedRunToDTO(run store.ListRunsByRepositoryRow) dtos.RunDTO {
	var completedAt *time.Time
	if run.CompletedAt.Valid {
		completedAt = &run.CompletedAt.Time
	}

	return dtos.RunDTO{
		ID:              run.ID,
		Prompt:          run.Prompt,
		FeatureID:       run.FeatureID,
		Status:          dtos.RunStatus(run.Status),
		CreatedByUserID: run.CreatedByUserID,
		CreatedAt:       run.CreatedAt.Time,
		CompletedAt:     completedAt,
		PRURL:           run.PrUrl,
	}
}

func CreateSandboxDir(workspace string, organizationID int64, repositoryID int64) (string, error) {
	workspaceTrimmed := strings.TrimSuffix(workspace, "/")
	dir := filepath.Join(workspaceTrimmed, fmt.Sprintf("%d/%d", organizationID, repositoryID))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *RunService) CreateRun(ctx context.Context, p CreateRunParams) (*dtos.RunDTO, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
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

	// Advance feature build status to in_progress when a run is kicked off.
	feature, err := qtx.GetFeature(ctx, p.FeatureID)
	if err == nil && (!feature.BuildStatus.Valid || feature.BuildStatus.BuildStatusType == store.BuildStatusTypePending) {
		_ = qtx.SetFeatureBuildStatus(ctx, store.SetFeatureBuildStatusParams{
			BuildStatus: store.NullBuildStatusType{BuildStatusType: store.BuildStatusTypeInProgress, Valid: true},
			ID:          p.FeatureID,
		})
	}

	if _, err := s.jc.InsertTx(ctx, tx, args.CloneRepoArgs{RunID: run.ID}, nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.hub.Publish(repo.ID, hub.EventFeatureUpdated)
	s.hub.Publish(repo.ID, hub.EventRunUpdated)

	return storeToRunDTO(run), nil
}

func (s *RunService) GetRunByID(ctx context.Context, runID int64) (store.GetRunByIDRow, error) {
	return s.q.GetRunByID(ctx, runID)
}

func (s *RunService) ListRunsByRepository(ctx context.Context, repositoryID int64) ([]dtos.RunDTO, error) {
	runs, err := s.q.ListRunsByRepository(ctx, repositoryID)
	if err != nil {
		return nil, err
	}

	out := make([]dtos.RunDTO, len(runs))
	for i, run := range runs {
		out[i] = listedRunToDTO(run)
	}
	return out, nil
}

func (s *RunService) updateRunStatus(ctx context.Context, runID int64, repoID int64, status store.FeatureRunStatus) error {
	// Job contexts can be cancelled/timed out; status updates should still be attempted.
	statusCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.q.UpdateRunStatus(statusCtx, store.UpdateRunStatusParams{
		Status: status,
		ID:     runID,
	}); err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	s.hub.Publish(repoID, hub.EventRunUpdated)
	return nil
}

//nolint:gosec // Git subprocess arguments use server-managed clone dirs and internal branch names only.
func prepareWorktree(ctx context.Context, repoDir, worktreeDir, branch, authURL string) error {
	// Refresh the remote URL with a current installation token before fetching —
	// tokens expire after 1 hour and the clone may have happened much earlier.
	setURL := exec.CommandContext(ctx, "git", "-C", repoDir, "remote", "set-url", "origin", authURL)
	if out, err := setURL.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote set-url: %w: %s", err, out)
	}

	fetch := exec.CommandContext(ctx, "git", "-C", repoDir, "fetch", "origin", "main")
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch: %w: %s", err, out)
	}

	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0750); err != nil {
		return err
	}

	// Make this idempotent across job retries:
	// - If an earlier attempt created the worktree or branch, remove/reset and recreate.
	_ = exec.CommandContext(ctx, "git", "-C", repoDir, "worktree", "remove", "--force", worktreeDir).Run()
	_ = os.RemoveAll(worktreeDir)

	// Ensure the branch points at the expected base (origin/main). This avoids failures when a
	// retry sees an existing local branch name.
	resetBranch := exec.CommandContext(ctx, "git", "-C", repoDir, "branch", "-f", branch, "origin/main")
	if out, err := resetBranch.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -f: %w: %s", err, out)
	}

	add := exec.CommandContext(ctx, "git", "-C", repoDir, "worktree", "add", worktreeDir, branch)
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return nil
}

//nolint:gosec // Git runs inside an isolated worktree; commit message is truncated server-side metadata.
func commitWorktreeChanges(ctx context.Context, worktreeDir, prompt string) error {
	add := exec.CommandContext(ctx, "git", "-C", worktreeDir, "add", "-A")
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}

	// Check if there is anything to commit before attempting the commit.
	diff := exec.CommandContext(ctx, "git", "-C", worktreeDir, "diff", "--cached", "--quiet")
	if err := diff.Run(); err == nil {
		// Exit 0 means no staged changes — nothing to commit.
		return nil
	}

	msg := prompt
	if len(msg) > 72 {
		msg = msg[:72]
	}
	commit := exec.CommandContext(ctx, "git", "-C", worktreeDir,
		"-c", "user.name=vote-llm agent",
		"-c", "user.email=agent@vote-llm",
		"commit", "-m", msg,
	)
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

func (s *RunService) RunAgent(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		return err
	}

	if run.Status == store.FeatureRunStatusCancelled {
		return nil
	}

	if err := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusRunning); err != nil {
		return err
	}

	repoDir := filepath.Join(run.Workspace, run.RepositoryName)
	worktreeDir := filepath.Join(run.Workspace, "worktrees", fmt.Sprintf("run-%d", run.ID))
	branch := fmt.Sprintf("feature-%d-run-%d", run.FeatureID, run.ID)

	authURL, err := s.githubSvc.AuthenticatedRepoURL(ctx, run.OrganizationID, run.RepositoryOwner, run.RepositoryName)
	if err != nil {
		if statusErr := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusFailed); statusErr != nil {
			return statusErr
		}
		return fmt.Errorf("failed to get authenticated repo URL: %w", err)
	}

	if err := prepareWorktree(ctx, repoDir, worktreeDir, branch, authURL); err != nil {
		if statusErr := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusFailed); statusErr != nil {
			return statusErr
		}
		return fmt.Errorf("failed to prepare worktree: %w", err)
	}

	onStart := func(pid int) {
		pid32 := int32(pid)
		storeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = s.q.UpdateRunPID(storeCtx, store.UpdateRunPIDParams{Pid: &pid32, ID: runID})
	}

	if err := s.runner.Run(ctx, run.Prompt, worktreeDir, onStart); err != nil {
		if statusErr := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusFailed); statusErr != nil {
			return statusErr
		}
		return fmt.Errorf("failed to run agent: %w", err)
	}

	if err := commitWorktreeChanges(ctx, worktreeDir, run.Prompt); err != nil {
		if statusErr := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusFailed); statusErr != nil {
			return statusErr
		}
		return fmt.Errorf("failed to commit agent changes: %w", err)
	}

	if _, err := s.jc.Insert(ctx, args.OpenRepoPullRequestArgs{
		OrganizationID: run.OrganizationID,
		RunID:          runID,
		Owner:          run.RepositoryOwner,
		Name:           run.RepositoryName,
		BranchName:     branch,
		WorktreeDir:    worktreeDir,
		Prompt:         run.Prompt,
	}, nil); err != nil {
		return fmt.Errorf("failed to enqueue open PR job: %w", err)
	}

	if err := s.updateRunStatus(ctx, runID, run.RepositoryID, store.FeatureRunStatusCompleted); err != nil {
		return err
	}

	return nil
}

func (s *RunService) DeleteRun(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}
	if run.Status != store.FeatureRunStatusCancelled && run.Status != store.FeatureRunStatusFailed {
		return ErrRunNotDeletable
	}
	if err := s.q.DeleteCancelledRun(ctx, runID); err != nil {
		return err
	}
	s.hub.Publish(run.RepositoryID, hub.EventRunUpdated)
	return nil
}

func (s *RunService) SetRunPRURL(ctx context.Context, runID int64, prURL string) error {
	return s.q.UpdateRunPRURL(ctx, store.UpdateRunPRURLParams{PrUrl: &prURL, ID: runID})
}

func (s *RunService) CancelRun(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}

	switch run.Status {
	case store.FeatureRunStatusPending:
		if err := s.q.SetRunCancelled(ctx, runID); err != nil {
			return err
		}
		s.hub.Publish(run.RepositoryID, hub.EventRunUpdated)
		return nil

	case store.FeatureRunStatusRunning:
		if run.Pid != nil {
			//nolint:gosec // PID is stored by the server from cmd.Process.Pid; not user-controlled.
			_ = syscall.Kill(int(*run.Pid), syscall.SIGTERM)
		}
		if err := s.q.SetRunCancelled(ctx, runID); err != nil {
			return err
		}
		s.hub.Publish(run.RepositoryID, hub.EventRunUpdated)
		return nil

	default:
		return ErrRunNotCancellable
	}
}
