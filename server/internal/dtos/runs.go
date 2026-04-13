package dtos

import "time"

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

type RunDTO struct {
	ID              int64      `json:"id"`
	Prompt          string     `json:"prompt"`
	FeatureID       int64      `json:"feature_id"`
	Status          RunStatus  `json:"status"`
	CreatedByUserID int64      `json:"created_by_user_id"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}
