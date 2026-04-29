package gitauth

import (
	gitauth_account "github.com/didrikolofsson/github-vote-llm/internal/gitauth/account"
	gitauth_app "github.com/didrikolofsson/github-vote-llm/internal/gitauth/app"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
)

type GitAuthClient interface {
	NewAccountClient(clientID string) *gitauth_account.GithubAccountClient
	NewAppClient() *gitauth_app.GithubAppClient
}

type GitAuthClientImpl struct {
	q         *store.Queries
	jwtSecret string
}

func New(q *store.Queries, clientID string, jwtSecret string) GitAuthClient {
	return &GitAuthClientImpl{
		q: q, jwtSecret: jwtSecret,
	}
}

func (c *GitAuthClientImpl) NewAccountClient(clientID string) *gitauth_account.GithubAccountClient {
	return gitauth_account.New(c.q, clientID, c.jwtSecret)
}

func (c *GitAuthClientImpl) NewAppClient() *gitauth_app.GithubAppClient {
	return gitauth_app.New()
}
