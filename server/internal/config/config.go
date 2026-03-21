// Hardcoded config values
package config

import "time"

// DefaultWorkspaceDir is the fallback workspace directory when WORKSPACE_DIR is not set.
const DefaultWorkspaceDir = "/tmp/vote-llm-workspaces"

// Hardcoded agent defaults.
const (
	AgentCommand        = "claude"
	AgentMaxTurns       = 25
	AgentMaxBudgetUSD   = 5.00
	AgentTimeoutMinutes = 30

	// Label names.
	LabelApproved       = "approved-for-dev"
	LabelFeatureRequest = "feature-request"
	LabelInProgress     = "llm-in-progress"
	LabelDone           = "llm-pr-created"
	LabelFailed         = "llm-failed"

	// Auth
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
	AuthCodeTTL     = 5 * time.Minute
)
