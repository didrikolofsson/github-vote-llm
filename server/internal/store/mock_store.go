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
	ListExecutionsFn                     func(ctx context.Context, limit, offset int32) ([]*ExecutionModel, error)
	GetExecutionByIDFn                   func(ctx context.Context, id int64) (*ExecutionModel, error)
	CancelExecutionFn                    func(ctx context.Context, id int64) (*ExecutionModel, error)
	RetryExecutionFn                     func(ctx context.Context, id int64) (*ExecutionModel, error)
	ListRepoConfigsFn                    func(ctx context.Context) ([]*RepoConfigModel, error)
	UpsertRepoConfigFn                   func(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfigModel, error)
	DeleteRepoConfigFn                   func(ctx context.Context, owner, repo string) error
	ListProposalsFn                      func(ctx context.Context, owner, repo string) ([]*ProposalModel, error)
	GetProposalFn                        func(ctx context.Context, id int64) (*ProposalModel, error)
	CreateProposalFn                     func(ctx context.Context, owner, repo, title, description string) (*ProposalModel, error)
	IncrementProposalVoteFn              func(ctx context.Context, id int64) (*ProposalModel, error)
	UpdateProposalStatusFn               func(ctx context.Context, id int64, status string) (*ProposalModel, error)
	ListProposalCommentsFn               func(ctx context.Context, proposalID int64) ([]*ProposalCommentModel, error)
	CreateProposalCommentFn              func(ctx context.Context, proposalID int64, body, authorName string) (*ProposalCommentModel, error)
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

func (m *MockStore) ListExecutions(ctx context.Context, limit, offset int32) ([]*ExecutionModel, error) {
	if m.ListExecutionsFn == nil {
		return nil, nil
	}
	return m.ListExecutionsFn(ctx, limit, offset)
}

func (m *MockStore) GetExecutionByID(ctx context.Context, id int64) (*ExecutionModel, error) {
	if m.GetExecutionByIDFn == nil {
		return nil, nil
	}
	return m.GetExecutionByIDFn(ctx, id)
}

func (m *MockStore) CancelExecution(ctx context.Context, id int64) (*ExecutionModel, error) {
	if m.CancelExecutionFn == nil {
		return nil, nil
	}
	return m.CancelExecutionFn(ctx, id)
}

func (m *MockStore) RetryExecution(ctx context.Context, id int64) (*ExecutionModel, error) {
	if m.RetryExecutionFn == nil {
		return nil, nil
	}
	return m.RetryExecutionFn(ctx, id)
}

func (m *MockStore) ListRepoConfigs(ctx context.Context) ([]*RepoConfigModel, error) {
	if m.ListRepoConfigsFn == nil {
		return nil, nil
	}
	return m.ListRepoConfigsFn(ctx)
}

func (m *MockStore) UpsertRepoConfig(ctx context.Context, params UpsertRepoConfigParams) (*RepoConfigModel, error) {
	if m.UpsertRepoConfigFn == nil {
		return nil, nil
	}
	return m.UpsertRepoConfigFn(ctx, params)
}

func (m *MockStore) DeleteRepoConfig(ctx context.Context, owner, repo string) error {
	if m.DeleteRepoConfigFn == nil {
		return nil
	}
	return m.DeleteRepoConfigFn(ctx, owner, repo)
}

func (m *MockStore) ListProposals(ctx context.Context, owner, repo string) ([]*ProposalModel, error) {
	if m.ListProposalsFn == nil {
		return nil, nil
	}
	return m.ListProposalsFn(ctx, owner, repo)
}

func (m *MockStore) GetProposal(ctx context.Context, id int64) (*ProposalModel, error) {
	if m.GetProposalFn == nil {
		return nil, nil
	}
	return m.GetProposalFn(ctx, id)
}

func (m *MockStore) CreateProposal(ctx context.Context, owner, repo, title, description string) (*ProposalModel, error) {
	if m.CreateProposalFn == nil {
		return nil, nil
	}
	return m.CreateProposalFn(ctx, owner, repo, title, description)
}

func (m *MockStore) IncrementProposalVote(ctx context.Context, id int64) (*ProposalModel, error) {
	if m.IncrementProposalVoteFn == nil {
		return nil, nil
	}
	return m.IncrementProposalVoteFn(ctx, id)
}

func (m *MockStore) UpdateProposalStatus(ctx context.Context, id int64, status string) (*ProposalModel, error) {
	if m.UpdateProposalStatusFn == nil {
		return nil, nil
	}
	return m.UpdateProposalStatusFn(ctx, id, status)
}

func (m *MockStore) ListProposalComments(ctx context.Context, proposalID int64) ([]*ProposalCommentModel, error) {
	if m.ListProposalCommentsFn == nil {
		return nil, nil
	}
	return m.ListProposalCommentsFn(ctx, proposalID)
}

func (m *MockStore) CreateProposalComment(ctx context.Context, proposalID int64, body, authorName string) (*ProposalCommentModel, error) {
	if m.CreateProposalCommentFn == nil {
		return nil, nil
	}
	return m.CreateProposalCommentFn(ctx, proposalID, body, authorName)
}
