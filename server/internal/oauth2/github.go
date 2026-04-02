package oauth2

import (
	"context"
	"encoding/base64"

	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	oa "golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	GitHubAuthURL  = "https://github.com/login/oauth/authorize"
	GitHubTokenURL = "https://github.com/login/oauth/access_token"
)

func NewGitHubOAuthConfig(clientID, clientSecret, redirectURL string) *oa.Config {
	return &oa.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email"},
	}
}

type GithubTokenSource struct {
	userID             int64
	q                  *store.Queries
	config             *oa.Config
	tokenEncryptionKey string
}

func NewGithubTokenSource(userID int64, q *store.Queries, config *oa.Config, tokenEncryptionKey string) *GithubTokenSource {
	return &GithubTokenSource{userID: userID, q: q, config: config, tokenEncryptionKey: tokenEncryptionKey}
}

// This function is called each time a new access token is needed
func (s *GithubTokenSource) Token() (*oa.Token, error) {
	// Get access and refresh tokens from DB
	connection, err := s.q.GetGitHubConnectionByUserID(context.Background(), s.userID)
	if err != nil {
		return nil, err
	}

	// Decrypt access token
	accessTokenBytes, err := base64.StdEncoding.DecodeString(connection.AccessTokenEncrypted)
	if err != nil {
		return nil, err
	}
	decrypted, err := encryption.Decrypt(accessTokenBytes, s.tokenEncryptionKey)
	if err != nil {
		return nil, err
	}
	accessToken := string(decrypted)

	// Decrypt refresh token
	var refreshToken string
	if connection.RefreshToken != nil {
		refreshTokenBytes, err := base64.StdEncoding.DecodeString(*connection.RefreshToken)
		if err != nil {
			return nil, err
		}
		decrypted, err := encryption.Decrypt(refreshTokenBytes, s.tokenEncryptionKey)
		if err != nil {
			return nil, err
		}
		refreshToken = string(decrypted)
	}

	token := &oa.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       connection.TokenExpiresAt.Time,
	}

	// If token is valid, return it
	if token.Valid() {
		return token, nil
	}

	// Refresh token if it's not valid
	token, err = s.config.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, err
	}

	// Encrypt new access token
	var encryptedAccessToken string
	encryptedAccessTokenBytes, err := encryption.Encrypt([]byte(token.AccessToken), s.tokenEncryptionKey)
	if err != nil {
		return nil, err
	}
	encryptedAccessToken = base64.StdEncoding.EncodeToString(encryptedAccessTokenBytes)

	// Encrypt new refresh token
	var encryptedRefreshToken *string
	if token.RefreshToken != "" {
		encrypted, err := encryption.Encrypt([]byte(token.RefreshToken), s.tokenEncryptionKey)
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(encrypted)
		encryptedRefreshToken = &encoded
	}

	_, err = s.q.UpsertGitHubConnection(context.Background(), store.UpsertGitHubConnectionParams{
		UserID:               s.userID,
		AccessTokenEncrypted: encryptedAccessToken,
		RefreshToken:         encryptedRefreshToken,
		TokenExpiresAt:       pgtype.Timestamptz{Time: token.Expiry, Valid: true},
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
