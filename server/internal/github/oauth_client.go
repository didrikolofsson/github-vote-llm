package github

import (
	"context"

	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// OAuthClient is a GitHub API client that uses a user's OAuth access token.
type OAuthClient struct {
	client *gh.Client
}

// NewOAuthClient creates a client that uses the given access token.
func NewOAuthClient(accessToken string) *OAuthClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &OAuthClient{client: gh.NewClient(httpClient)}
}

// RepoSummary is a minimal repo representation for listing.
type RepoSummary struct {
	Owner string
	Repo  string
}

// ListRepos returns repositories the user has access to (user + org repos).
func (c *OAuthClient) ListRepos(ctx context.Context, page int) ([]RepoSummary, bool, error) {
	const perPage = 30
	opts := &gh.RepositoryListOptions{
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gh.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := c.client.Repositories.List(ctx, "", opts)
	if err != nil {
		return nil, false, err
	}
	out := make([]RepoSummary, len(repos))
	for i, r := range repos {
		owner := r.GetOwner().GetLogin()
		name := r.GetName()
		out[i] = RepoSummary{Owner: owner, Repo: name}
	}
	hasMore := resp.NextPage != 0
	return out, hasMore, nil
}
