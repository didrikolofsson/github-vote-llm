package api_services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
)

// UpdateConfigInput holds the optional fields for updating a repo config.
type UpdateConfigInput struct {
	LabelApproved       *string
	LabelInProgress     *string
	LabelDone           *string
	LabelFailed         *string
	LabelFeatureRequest *string
	VoteThreshold       *int32
	TimeoutMinutes      *int32
	MaxBudgetUsd        *float64
	AnthropicAPIKey     *string
	IsBoardPublic       *bool
}

// ReposService handles business logic for repo configuration operations.
type ReposService struct {
	store store.Store
}

// NewReposService creates a new ReposService.
func NewReposService(s store.Store) *ReposService {
	return &ReposService{store: s}
}

// List returns all repo configurations.
func (s *ReposService) List(ctx context.Context) ([]*store.RepoConfigModel, error) {
	return s.store.ListRepoConfigs(ctx)
}

// GetConfig returns the resolved config for a repo. Returns nil if no row exists.
func (s *ReposService) GetConfig(ctx context.Context, owner, repo string) (*store.RepoConfigModel, error) {
	return s.store.GetRepoConfig(ctx, owner, repo)
}

// DeleteConfig removes the repo configuration.
func (s *ReposService) DeleteConfig(ctx context.Context, owner, repo string) error {
	return s.store.DeleteRepoConfig(ctx, owner, repo)
}

// ListProposals returns all proposals for a repo, sorted by vote count.
func (s *ReposService) ListProposals(ctx context.Context, owner, repo string) ([]*store.ProposalModel, error) {
	return s.store.ListProposals(ctx, owner, repo)
}

// UpdateProposalStatus updates the status of a proposal.
func (s *ReposService) UpdateProposalStatus(ctx context.Context, id int64, status string) (*store.ProposalModel, error) {
	return s.store.UpdateProposalStatus(ctx, id, status)
}

// UpdateConfig upserts the repo configuration.
func (s *ReposService) UpdateConfig(ctx context.Context, owner, repo string, in UpdateConfigInput) (*store.RepoConfigModel, error) {
	isBoardPublic := false
	if in.IsBoardPublic != nil {
		isBoardPublic = *in.IsBoardPublic
	}
	params := store.UpsertRepoConfigParams{
		Owner:               owner,
		Repo:                repo,
		LabelApproved:       in.LabelApproved,
		LabelInProgress:     in.LabelInProgress,
		LabelDone:           in.LabelDone,
		LabelFailed:         in.LabelFailed,
		LabelFeatureRequest: in.LabelFeatureRequest,
		VoteThreshold:       in.VoteThreshold,
		TimeoutMinutes:      in.TimeoutMinutes,
		MaxBudgetUsd:        helpers.Float64ToNumeric(in.MaxBudgetUsd),
		AnthropicApiKey:     in.AnthropicAPIKey,
		IsBoardPublic:       isBoardPublic,
	}
	return s.store.UpsertRepoConfig(ctx, params)
}
