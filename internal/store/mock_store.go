package store

import "context"

// MockStore is a test double for Store. Each method delegates to its corresponding
// Fn field, which tests can set to control behavior and capture calls.
type MockStore struct {
	GetExecutionByOwnerRepoIssueNumberFn func(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	CreateExecutionFn                    func(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	ResetFailedExecutionFn               func(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error)
	ResetExecutionFn                     func(ctx context.Context, id int64) (*ExecutionModel, error)
	SetInProgressFn                      func(ctx context.Context, id int64, branch string) (*ExecutionModel, error)
	SetSuccessFn                         func(ctx context.Context, id int64, prURL string) (*ExecutionModel, error)
	SetFailedFn                          func(ctx context.Context, id int64, errMsg string) (*ExecutionModel, error)
	GetRepoConfigFn                      func(ctx context.Context, owner, repo string) (*RepoConfigModel, error)
	IncrementIssueVoteFn                 func(ctx context.Context, owner, repo string, issueNumber int) (*IssueVoteModel, error)
}

var _ Store = (*MockStore)(nil)

func (m *MockStore) GetExecutionByOwnerRepoIssueNumber(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	return m.GetExecutionByOwnerRepoIssueNumberFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	return m.CreateExecutionFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) ResetFailedExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionModel, error) {
	return m.ResetFailedExecutionFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) SetInProgress(ctx context.Context, id int64, branch string) (*ExecutionModel, error) {
	return m.SetInProgressFn(ctx, id, branch)
}

func (m *MockStore) SetSuccess(ctx context.Context, id int64, prURL string) (*ExecutionModel, error) {
	return m.SetSuccessFn(ctx, id, prURL)
}

func (m *MockStore) SetFailed(ctx context.Context, id int64, errMsg string) (*ExecutionModel, error) {
	return m.SetFailedFn(ctx, id, errMsg)
}

func (m *MockStore) GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfigModel, error) {
	if m.GetRepoConfigFn == nil {
		return nil, nil
	}
	return m.GetRepoConfigFn(ctx, owner, repo)
}

func (m *MockStore) IncrementIssueVote(ctx context.Context, owner, repo string, issueNumber int) (*IssueVoteModel, error) {
	if m.IncrementIssueVoteFn == nil {
		return &IssueVoteModel{VoteCount: 1}, nil
	}
	return m.IncrementIssueVoteFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) ResetExecution(ctx context.Context, id int64) (*ExecutionModel, error) {
	return m.ResetExecutionFn(ctx, id)
}
