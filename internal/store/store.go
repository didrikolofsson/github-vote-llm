package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAlreadyExists is returned by CreateExecution when a row for the same
// (owner, repo, issue_number) already exists.
var ErrAlreadyExists = errors.New("execution already exists for this issue")

// Store is the interface for all database operations used by this service.
type Store interface {
	CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error)
	SetInProgress(ctx context.Context, id int64, branch string) (*Execution, error)
	SetSuccess(ctx context.Context, id int64, prURL string) (*Execution, error)
	SetFailed(ctx context.Context, id int64, errMsg string) (*Execution, error)
	GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfig, error)
	UpsertRepoConfig(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfig, error)
}

// compile-time interface check
var _ Store = (*PostgresStore)(nil)

// PostgresStore implements Store backed by a pgxpool.Pool.
type PostgresStore struct {
	q *Queries
}

// NewPostgresStore creates a Store backed by the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{q: New(pool)}
}

func (s *PostgresStore) CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error) {
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
	return &exec, nil
}

func (s *PostgresStore) SetInProgress(ctx context.Context, id int64, branch string) (*Execution, error) {
	exec, err := s.q.UpdateExecutionInProgress(ctx, UpdateExecutionInProgressParams{
		Branch: &branch,
		ID:     id,
	})
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *PostgresStore) SetSuccess(ctx context.Context, id int64, prURL string) (*Execution, error) {
	exec, err := s.q.UpdateExecutionSuccess(ctx, UpdateExecutionSuccessParams{
		PrUrl: &prURL,
		ID:    id,
	})
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *PostgresStore) SetFailed(ctx context.Context, id int64, errMsg string) (*Execution, error) {
	exec, err := s.q.UpdateExecutionFailed(ctx, UpdateExecutionFailedParams{
		Error: &errMsg,
		ID:    id,
	})
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *PostgresStore) GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfig, error) {
	cfg, err := s.q.GetRepoConfig(ctx, GetRepoConfigParams{Owner: owner, Repo: repo})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *PostgresStore) UpsertRepoConfig(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfig, error) {
	cfg, err := s.q.UpsertRepoConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
