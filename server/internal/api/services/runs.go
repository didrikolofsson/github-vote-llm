package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
)

// RunsService handles business logic for execution (run) operations.
type RunsService struct {
	store store.Store
}

// NewRunsService creates a new RunsService.
func NewRunsService(s store.Store) *RunsService {
	return &RunsService{store: s}
}

// List returns a paginated list of executions.
func (s *RunsService) List(ctx context.Context, limit, offset int32) ([]*store.ExecutionModel, error) {
	return s.store.ListExecutions(ctx, limit, offset)
}

// Get returns a single execution by ID. Returns nil if not found.
func (s *RunsService) Get(ctx context.Context, id int64) (*store.ExecutionModel, error) {
	return s.store.GetExecutionByID(ctx, id)
}

// Create creates a new pending execution for the given issue.
// Returns store.ErrAlreadyExists if one already exists.
func (s *RunsService) Create(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
	return s.store.CreateExecution(ctx, owner, repo, issueNumber)
}

// Retry resets a failed or cancelled execution back to pending.
func (s *RunsService) Retry(ctx context.Context, id int64) (*store.ExecutionModel, error) {
	return s.store.RetryExecution(ctx, id)
}

// Cancel cancels a pending or in-progress execution.
func (s *RunsService) Cancel(ctx context.Context, id int64) (*store.ExecutionModel, error) {
	return s.store.CancelExecution(ctx, id)
}
