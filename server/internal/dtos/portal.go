package dtos

import "github.com/didrikolofsson/github-vote-llm/internal/store"

// PortalFeatureDTO is the public-facing shape for a feature card.
type PortalFeatureDTO struct {
	ID           int64                  `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	ReviewStatus store.ReviewStatusType `json:"review_status"`
	BuildStatus  *store.BuildStatusType `json:"build_status"`
	Area         *string                `json:"area"`
	VoteCount    int64                  `json:"vote_count"`
	HasVoted     bool                   `json:"has_voted"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// PortalCommentDTO is the public-facing shape for a comment.
type PortalCommentDTO struct {
	ID         int64  `json:"id"`
	FeatureID  int64  `json:"feature_id"`
	Body       string `json:"body"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

// PortalPageDTO is the full data payload for a public portal page.
type PortalPageDTO struct {
	OrgSlug    string             `json:"org_slug"`
	RepoOwner  string             `json:"repo_owner"`
	RepoName   string             `json:"repo_name"`
	RepoID     int64              `json:"repo_id"`
	Requests   []PortalFeatureDTO `json:"requests"`    // build_status: NULL (approved, not yet committed)
	Pending    []PortalFeatureDTO `json:"pending"`     // build_status: pending (committed, not started)
	InProgress []PortalFeatureDTO `json:"in_progress"` // build_status: in_progress or stuck
	Done       []PortalFeatureDTO `json:"done"`        // build_status: done
}
