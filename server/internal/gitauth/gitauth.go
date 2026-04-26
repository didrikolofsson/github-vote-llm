package gitauth

import (
	gitauth_app "github.com/didrikolofsson/github-vote-llm/internal/gitauth/app"
	gitauth_oauth "github.com/didrikolofsson/github-vote-llm/internal/gitauth/oauth"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GitAuthClient interface {
	NewOauthClient(clientID string) *gitauth_oauth.GithubOauthClient
	NewAppClient(db *pgxpool.Pool) *gitauth_app.GithubAppClient
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

func (c *GitAuthClientImpl) NewOauthClient(clientID string) *gitauth_oauth.GithubOauthClient {
	return gitauth_oauth.New(c.q, clientID, c.jwtSecret)
}

func (c *GitAuthClientImpl) NewAppClient(db *pgxpool.Pool) *gitauth_app.GithubAppClient {
	return gitauth_app.New(db)
}
