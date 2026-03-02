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
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
