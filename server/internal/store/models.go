package store

import (
	"time"
)

type RepoConfigModel struct {
	ID                  int64
	Owner               string
	Repo                string
	LabelApproved       string
	LabelInProgress     string
	LabelDone           string
	LabelFailed         string
	LabelFeatureRequest string
	VoteThreshold       int32
	TimeoutMinutes      int32
	MaxBudgetUsd        float64
	AnthropicAPIKey     string
	IsBoardPublic       bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ProposalModel struct {
	ID          int64
	Owner       string
	Repo        string
	Title       string
	Description string
	VoteCount   int32
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProposalCommentModel struct {
	ID         int64
	ProposalID int64
	Body       string
	AuthorName string
	CreatedAt  time.Time
}

type ExecutionModel struct {
	ID          int64
	Owner       string
	Repo        string
	IssueNumber int32
	Status      string
	Branch      *string
	PrUrl       *string
	Error       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type IssueVoteModel struct {
	ID          int64
	Owner       string
	Repo        string
	IssueNumber int32
	VoteCount   int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
