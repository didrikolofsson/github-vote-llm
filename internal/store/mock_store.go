package store

import "context"

// MockStore is a test double for Store. Each method delegates to its corresponding
// Fn field, which tests can set to control behavior and capture calls.
type MockStore struct {
	GetExecutionByOwnerRepoIssueNumberFn func(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error)
	CreateExecutionFn                    func(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error)
	ResetFailedExecutionFn               func(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error)
	ResetExecutionFn                     func(ctx context.Context, id int64) (*Execution, error)
	SetInProgressFn                      func(ctx context.Context, id int64, branch string) (*Execution, error)
	SetSuccessFn                         func(ctx context.Context, id int64, prURL string) (*Execution, error)
	SetFailedFn                          func(ctx context.Context, id int64, errMsg string) (*Execution, error)
	GetRepoConfigFn                      func(ctx context.Context, owner, repo string) (*RepoConfig, error)
	UpsertRepoConfigFn                   func(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfig, error)
}

var _ Store = (*MockStore)(nil)

func (m *MockStore) GetExecutionByOwnerRepoIssueNumber(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error) {
	return m.GetExecutionByOwnerRepoIssueNumberFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) CreateExecution(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error) {
	return m.CreateExecutionFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) ResetFailedExecution(ctx context.Context, owner, repo string, issueNumber int) (*Execution, error) {
	return m.ResetFailedExecutionFn(ctx, owner, repo, issueNumber)
}

func (m *MockStore) SetInProgress(ctx context.Context, id int64, branch string) (*Execution, error) {
	return m.SetInProgressFn(ctx, id, branch)
}

func (m *MockStore) SetSuccess(ctx context.Context, id int64, prURL string) (*Execution, error) {
	return m.SetSuccessFn(ctx, id, prURL)
}

func (m *MockStore) SetFailed(ctx context.Context, id int64, errMsg string) (*Execution, error) {
	return m.SetFailedFn(ctx, id, errMsg)
}

func (m *MockStore) GetRepoConfig(ctx context.Context, owner, repo string) (*RepoConfig, error) {
	if m.GetRepoConfigFn == nil {
		return nil, nil
	}
	return m.GetRepoConfigFn(ctx, owner, repo)
}

func (m *MockStore) UpsertRepoConfig(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfig, error) {
	return m.UpsertRepoConfigFn(ctx, params)
}

func (m *MockStore) ResetExecution(ctx context.Context, id int64) (*Execution, error) {
	return m.ResetExecutionFn(ctx, id)
}
