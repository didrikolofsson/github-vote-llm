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

type GithubAccountType string

const (
	GithubAccountTypeUser         GithubAccountType = "user"
	GithubAccountTypeOrganization GithubAccountType = "organization"
)

type AppInstallationStatus struct {
	Installed           bool              `json:"installed"`
	SuspendedAt         *time.Time        `json:"suspended_at,omitempty"`
	InstalledByUserName string            `json:"installed_by_user_name,omitempty"`
	TargetLogin         string            `json:"target_login,omitempty"`
	AccountType         GithubAccountType `json:"account_type,omitempty"`
}

type AppInstallation struct {
	ID                   int64      `json:"id"`
	OrganizationID       int64      `json:"organization_id"`
	GithubInstallationID int64      `json:"github_installation_id"`
	GithubAccountLogin   string     `json:"github_account_login"`
	GithubAccountID      int64      `json:"github_account_id"`
	GithubAccountType    string     `json:"github_account_type"`
	RepositorySelection  string     `json:"repository_selection"`
	SuspendedAt          *time.Time `json:"suspended_at,omitempty"`
	InstalledByUserID    *int64     `json:"installed_by_user_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	State                string     `json:"state"`
}
