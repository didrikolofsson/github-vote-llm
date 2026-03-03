package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
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

// UpdateConfig upserts the repo configuration.
func (s *ReposService) UpdateConfig(ctx context.Context, owner, repo string, in UpdateConfigInput) (*store.RepoConfigModel, error) {
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
		MaxBudgetUsd:        float64ToNumeric(in.MaxBudgetUsd),
		AnthropicApiKey:     in.AnthropicAPIKey,
	}
	return s.store.UpsertRepoConfig(ctx, params)
}

// float64ToNumeric converts *float64 to pgtype.Numeric.
func float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*f); err != nil {
		return pgtype.Numeric{}
	}
	return n
}
