// Hardcoded config values
package config

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
	LabelCandidate      = "candidate"
)
