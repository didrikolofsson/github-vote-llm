package github

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
	gh "golang.org/x/oauth2/github"
)

func NewGithubOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     gh.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email"},
	}
}

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

func NewGithubClientByUserID(
	ctx context.Context,
	ts *GithubTokenSource,
) *github.Client {
	httpClient := oauth2.NewClient(ctx, ts)
	client := github.NewClient(httpClient)
	return client
}
