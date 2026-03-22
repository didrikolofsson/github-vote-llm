package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// GitHub OAuth scopes: repo (full), read:org for org repos
const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
)

type GitHubConnectionStatus struct {
	Login *string
}

type GitHubOAuthService interface {
	BuildAuthorizeURL(redirectURI, state string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (*GitHubTokenResponse, error)
	StoreConnection(ctx context.Context, userID int64, tokens *GitHubTokenResponse, encryptionKey string) error
	GetDecryptedToken(ctx context.Context, userID int64, encryptionKey string) (string, error)
	GetConnectionStatus(ctx context.Context, userID int64) (*GitHubConnectionStatus, error)
}

type GitHubTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds, 0 = no expiry
	GithubUserID int64
	GithubLogin  string
}

type GitHubOAuthServiceImpl struct {
	clientID     string
	clientSecret string
	q            *store.Queries
}

func NewGitHubOAuthService(clientID, clientSecret string, q *store.Queries) GitHubOAuthService {
	return &GitHubOAuthServiceImpl{
		clientID:     clientID,
		clientSecret: clientSecret,
		q:            q,
	}
}

func (s *GitHubOAuthServiceImpl) BuildAuthorizeURL(redirectURI, state string) string {
	v := url.Values{}
	v.Set("client_id", s.clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", "repo read:org")
	v.Set("state", state)
	return githubAuthorizeURL + "?" + v.Encode()
}

type githubTokenAPIResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *GitHubOAuthServiceImpl) ExchangeCode(ctx context.Context, code, redirectURI string) (*GitHubTokenResponse, error) {
	reqBody := url.Values{}
	reqBody.Set("client_id", s.clientID)
	reqBody.Set("client_secret", s.clientSecret)
	reqBody.Set("code", code)
	reqBody.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", githubTokenURL, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tok githubTokenAPIResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("github token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("github returned no access_token: %s", string(body))
	}

	res := &GitHubTokenResponse{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		Scope:        tok.Scope,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
	}
	// Fetch GitHub user to get login and id
	if id, login, err := fetchGitHubUser(ctx, tok.AccessToken); err == nil {
		res.GithubUserID = id
		res.GithubLogin = login
	}
	return res, nil
}

type githubUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func fetchGitHubUser(ctx context.Context, accessToken string) (int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	var u githubUserResponse
	if err := json.Unmarshal(body, &u); err != nil {
		return 0, "", err
	}
	return u.ID, u.Login, nil
}

// StoreConnection encrypts the access token and stores it in github_connections.
func (s *GitHubOAuthServiceImpl) StoreConnection(ctx context.Context, userID int64, tokens *GitHubTokenResponse, encryptionKey string) error {
	encrypted, err := encryption.Encrypt([]byte(tokens.AccessToken), encryptionKey)
	if err != nil {
		return err
	}
	encryptedB64 := base64.StdEncoding.EncodeToString(encrypted)

	var refreshToken *string
	if tokens.RefreshToken != "" {
		refreshToken = &tokens.RefreshToken
	}
	var expiresAt pgtype.Timestamptz
	if tokens.ExpiresIn > 0 {
		expiresAt = pgtype.Timestamptz{Time: time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second), Valid: true}
	}
	var ghUserID *int64
	if tokens.GithubUserID != 0 {
		ghUserID = &tokens.GithubUserID
	}
	var ghLogin *string
	if tokens.GithubLogin != "" {
		ghLogin = &tokens.GithubLogin
	}

	_, err = s.q.UpsertGitHubConnection(ctx, store.UpsertGitHubConnectionParams{
		UserID:               userID,
		AccessTokenEncrypted: encryptedB64,
		RefreshToken:         refreshToken,
		TokenExpiresAt:       expiresAt,
		GithubUserID:         ghUserID,
		GithubLogin:          ghLogin,
	})
	return err
}

// GetConnectionStatus returns the GitHub connection status (login) without the token.
func (s *GitHubOAuthServiceImpl) GetConnectionStatus(ctx context.Context, userID int64) (*GitHubConnectionStatus, error) {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &GitHubConnectionStatus{Login: conn.GithubLogin}, nil
}

// GetDecryptedToken returns the decrypted access token for the user.
func (s *GitHubOAuthServiceImpl) GetDecryptedToken(ctx context.Context, userID int64, encryptionKey string) (string, error) {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	encrypted, err := base64.StdEncoding.DecodeString(conn.AccessTokenEncrypted)
	if err != nil {
		return "", err
	}
	plaintext, err := encryption.Decrypt(encrypted, encryptionKey)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
