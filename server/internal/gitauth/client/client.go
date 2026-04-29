package gitauth_client

import (
	"context"

	"github.com/google/go-github/v84/github"
	"golang.org/x/oauth2"
	gha "golang.org/x/oauth2/github"
)

type OauthConfigParams struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	RedirectURL  string
}

func NewOauthConfig(params OauthConfigParams) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		Scopes:       params.Scopes,
		Endpoint:     gha.Endpoint,
		RedirectURL:  params.RedirectURL,
	}
}

func NewGithubClient(ctx context.Context, ts oauth2.TokenSource) *github.Client {
	httpClient := oauth2.NewClient(ctx, ts)
	return github.NewClient(httpClient)
}
