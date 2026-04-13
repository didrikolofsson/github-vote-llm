package dtos

type GitHubRepository struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type GitHubRepositoryListResponse struct {
	Repositories []GitHubRepository `json:"repositories"`
	HasMore      bool               `json:"has_more"`
}
