package store

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAlreadyExists is returned by CreateExecution when an execution record for
// the given (owner, repo, issue_number) already exists in the database.
var ErrAlreadyExists = errors.New("execution already exists for this issue")

// Store is the interface for all database operations used by this service.
type Store interface {
	GetExecutionByOwnerRepoIssueNumber(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	ResetFailedExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	ResetExecution(ctx context.Context, id int64) (*ExecutionModel, error)
	SetInProgress(ctx context.Context, id int64, branch string) (*ExecutionModel, error)
	SetSuccess(ctx context.Context, id int64, prURL string) (*ExecutionModel, error)
	SetFailed(ctx context.Context, id int64, errMsg string) (*ExecutionModel, error)
	GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfigModel, error)
}

// PostgresStore implements Store backed by a pgxpool.Pool.
type PostgresStore struct {
	q *Queries
}

// NewPostgresStore creates a Store backed by the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{q: New(pool)}
}

func isNoRowsError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)
}

// ptrOr returns *p if non-nil, otherwise defaultVal.
func ptrOr[T any](p *T, defaultVal T) T {
	if p != nil {
		return *p
	}
	return defaultVal
}

// numericToFloat64 converts pgtype.Numeric to float64. Returns 0 if not valid.
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

// numericOr returns the float64 value of n if valid, otherwise defaultVal.
func numericOr(n pgtype.Numeric, defaultVal float64) float64 {
	if n.Valid {
		return numericToFloat64(n)
	}
	return defaultVal
}

func (s *PostgresStore) GetExecutionByOwnerRepoIssueNumber(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	exec, err := s.q.GetExecutionByOwnerRepoIssueNumber(ctx, GetExecutionByOwnerRepoIssueNumberParams{
		Owner:       owner,
		Repo:        repo,
		IssueNumber: int32(issueNumber),
	})
	if isNoRowsError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	exec, err := s.q.CreateExecution(ctx, CreateExecutionParams{
		Owner:       owner,
		Repo:        repo,
		IssueNumber: int32(issueNumber),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) ResetFailedExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	exec, err := s.q.ResetFailedExecution(ctx, ResetFailedExecutionParams{
		Owner:       owner,
		Repo:        repo,
		IssueNumber: int32(issueNumber),
	})
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) ResetExecution(ctx context.Context, id int64) (*ExecutionModel, error) {
	exec, err := s.q.ResetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) SetInProgress(ctx context.Context, id int64, branch string) (*ExecutionModel, error) {
	exec, err := s.q.UpdateExecutionInProgress(ctx, UpdateExecutionInProgressParams{
		Branch: &branch,
		ID:     id,
	})
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) SetSuccess(ctx context.Context, id int64, prURL string) (*ExecutionModel, error) {
	exec, err := s.q.UpdateExecutionSuccess(ctx, UpdateExecutionSuccessParams{
		PrUrl: &prURL,
		ID:    id,
	})
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

func (s *PostgresStore) SetFailed(ctx context.Context, id int64, errMsg string) (*ExecutionModel, error) {
	exec, err := s.q.UpdateExecutionFailed(ctx, UpdateExecutionFailedParams{
		Error: &errMsg,
		ID:    id,
	})
	if err != nil {
		return nil, err
	}
	return &ExecutionModel{
		ID:          exec.ID,
		Owner:       exec.Owner,
		Repo:        exec.Repo,
		IssueNumber: exec.IssueNumber,
		Status:      exec.Status,
		Branch:      exec.Branch,
		PrUrl:       exec.PrUrl,
		Error:       exec.Error,
		CreatedAt:   exec.CreatedAt.Time,
		UpdatedAt:   exec.UpdatedAt.Time,
	}, nil
}

// GetRepoConfig returns the repo config for the given owner and repo.
// If no config is found, it returns nil.
func (s *PostgresStore) GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfigModel, error) {
	var repoConfig RepoConfigModel

	cfg, err := s.q.GetRepoConfig(ctx, GetRepoConfigParams{Owner: owner, Repo: repo})
	if err != nil {
		if isNoRowsError(err) {
			return nil, nil
		}
		return nil, err
	}

	repoConfig.ID = cfg.ID
	repoConfig.Owner = cfg.Owner
	repoConfig.Repo = cfg.Repo
	repoConfig.LabelApproved = ptrOr(cfg.LabelApproved, config.LabelApproved)
	repoConfig.LabelInProgress = ptrOr(cfg.LabelInProgress, config.LabelInProgress)
	repoConfig.LabelDone = ptrOr(cfg.LabelDone, config.LabelDone)
	repoConfig.LabelFailed = ptrOr(cfg.LabelFailed, config.LabelFailed)
	repoConfig.VoteThreshold = ptrOr(cfg.VoteThreshold, int32(config.AgentMaxTurns))
	repoConfig.TimeoutMinutes = ptrOr(cfg.TimeoutMinutes, int32(config.AgentTimeoutMinutes))
	repoConfig.MaxBudgetUsd = numericOr(cfg.MaxBudgetUsd, config.AgentMaxBudgetUSD)
	repoConfig.AnthropicAPIKey = ptrOr(cfg.AnthropicApiKey, os.Getenv("ANTHROPIC_API_KEY"))
	repoConfig.CreatedAt = cfg.CreatedAt.Time
	repoConfig.UpdatedAt = cfg.UpdatedAt.Time

	return &repoConfig, nil
}

// func (s *PostgresStore) UpsertRepoConfig(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfigModel, error) {
// 	cfg, err := s.q.UpsertRepoConfig(ctx, params)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &RepoConfigModel{
// 		ID:              cfg.ID,
// 		Owner:           cfg.Owner,
// 		Repo:            cfg.Repo,
// 		LabelApproved:   ptrOr(cfg.LabelApproved, config.LabelApproved),
// 		LabelInProgress: ptrOr(cfg.LabelInProgress, config.LabelInProgress),
// 		LabelDone:       ptrOr(cfg.LabelDone, config.LabelDone),
// 		LabelFailed:     ptrOr(cfg.LabelFailed, config.LabelFailed),
// 		VoteThreshold:   ptrOr(cfg.VoteThreshold, int32(config.AgentMaxTurns)),
// 		TimeoutMinutes:  ptrOr(cfg.TimeoutMinutes, int32(config.AgentTimeoutMinutes)),
// 		MaxBudgetUsd:    numericOr(cfg.MaxBudgetUsd, config.AgentMaxBudgetUSD),
// 		AnthropicAPIKey: ptrOr(cfg.AnthropicApiKey, os.Getenv("ANTHROPIC_API_KEY")),
// 		CreatedAt:       cfg.CreatedAt.Time,
// 		UpdatedAt:       cfg.UpdatedAt.Time,
// 	}, nil
// }
