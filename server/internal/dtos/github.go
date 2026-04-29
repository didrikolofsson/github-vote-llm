package dtos

import "time"

type GitHubRepository struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type GitHubRepositoryListResponse struct {
	Repositories []GitHubRepository `json:"repositories"`
	HasMore      bool               `json:"has_more"`
}

type AppInstallURLResponse struct {
	InstallURL string `json:"install_url"`
}

type AppInstallationStatusResponse struct {
	Installed   bool       `json:"installed"`
	TargetLogin string     `json:"target_login,omitempty"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
}
