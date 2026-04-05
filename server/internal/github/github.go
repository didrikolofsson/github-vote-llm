package github

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
	goagh "golang.org/x/oauth2/github"
)

// NewOAuthConfig builds the standard GitHub OAuth2 client config (authorize + token URLs, scopes).
type NewGithubOAuthConfigParams struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func NewGithubOAuthConfig(p NewGithubOAuthConfigParams) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     goagh.Endpoint,
		RedirectURL:  p.RedirectURL,
		Scopes:       []string{"user:email"},
	}
}

// GithubTokenSource implements [oauth2.TokenSource]: it reads the user's tokens
// from the DB, refreshes via [oauth2.Config.TokenSource] when expired, and
// persists the new ciphertext. ctx is used for DB I/O and the refresh HTTP
// request; construct a new GithubTokenSource per inbound request (see
// [oauth2.NewClient]) so cancellation propagates.
type GithubTokenSource struct {
	ctx                context.Context
	userID             int64
	q                  *store.Queries
	config             *oauth2.Config
	tokenEncryptionKey string
}

func NewGithubTokenSource(
	ctx context.Context,
	q *store.Queries,
	config *oauth2.Config,
	userID int64,
	tokenEncryptionKey string,
) *GithubTokenSource {
	return &GithubTokenSource{
		ctx:                ctx,
		q:                  q,
		config:             config,
		userID:             userID,
		tokenEncryptionKey: tokenEncryptionKey,
	}
}

func (ts *GithubTokenSource) Token() (*oauth2.Token, error) {
	conn, err := ts.q.GetGitHubConnectionByUserID(ts.ctx, ts.userID)
	if err != nil {
		return nil, err
	}

	accessPlain, err := encryption.Decrypt(conn.AccessTokenEncrypted, ts.tokenEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("github token: access token: %w", err)
	}
	accessToken := string(accessPlain)

	var refreshToken string
	if conn.RefreshToken != nil {
		refreshPlain, rerr := encryption.Decrypt(*conn.RefreshToken, ts.tokenEncryptionKey)
		if rerr != nil {
			return nil, fmt.Errorf("github token: refresh token: %w", rerr)
		}
		refreshToken = string(refreshPlain)
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	if conn.TokenExpiresAt.Valid {
		token.Expiry = conn.TokenExpiresAt.Time
	}

	if token.Valid() {
		return token, nil
	}

	token, err = ts.config.TokenSource(ts.ctx, token).Token()
	if err != nil {
		return nil, err
	}

	encryptedAccess, err := encryption.Encrypt([]byte(token.AccessToken), ts.tokenEncryptionKey)
	if err != nil {
		return nil, err
	}

	var encryptedRefresh *string
	switch {
	case token.RefreshToken != "":
		s, encErr := encryption.Encrypt([]byte(token.RefreshToken), ts.tokenEncryptionKey)
		if encErr != nil {
			return nil, encErr
		}
		encryptedRefresh = &s
	case conn.RefreshToken != nil:
		// Refresh responses often omit refresh_token; x/oauth2 fills the in-memory
		// token, but if it were empty, keep the existing ciphertext so we never
		// NULL out the column in the upsert.
		encryptedRefresh = conn.RefreshToken
	default:
		encryptedRefresh = nil
	}

	var expiresAt pgtype.Timestamptz
	if !token.Expiry.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: token.Expiry, Valid: true}
	}

	_, err = ts.q.UpsertGitHubConnection(ts.ctx, store.UpsertGitHubConnectionParams{
		UserID:               ts.userID,
		AccessTokenEncrypted: encryptedAccess,
		RefreshToken:         encryptedRefresh,
		TokenExpiresAt:       expiresAt,
		GithubUserID:         conn.GithubUserID,
		GithubLogin:          conn.GithubLogin,
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

type NewGithubClientByUserIDParams struct {
	Context            context.Context
	Queries            *store.Queries
	Config             *oauth2.Config
	UserID             int64
	TokenEncryptionKey string
}

func NewGithubClientByUserID(p NewGithubClientByUserIDParams) *github.Client {
	ts := NewGithubTokenSource(
		p.Context,
		p.Queries,
		p.Config,
		p.UserID,
		p.TokenEncryptionKey,
	)
	httpClient := oauth2.NewClient(p.Context, ts)
	client := github.NewClient(httpClient)
	return client
}
