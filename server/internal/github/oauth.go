package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

var ErrNotConnected = errors.New("github account not connected")

const (
	authorizeURL = "https://github.com/login/oauth/authorize"
	tokenURL     = "https://github.com/login/oauth/access_token"
)

type ConnectionStatus struct {
	Login *string
}

type TokenResponse struct {
	AccessToken  string
	TokenType    string
	Scope        string
	RefreshToken string
	ExpiresIn    int // seconds, 0 = no expiry
	GithubUserID int64
	GithubLogin  string
}

type OAuthService interface {
	BuildAuthorizeURL(redirectURI, state string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)
	StoreConnection(ctx context.Context, userID int64, tokens *TokenResponse, encryptionKey string) error
	GetDecryptedToken(ctx context.Context, userID int64, encryptionKey string) (string, error)
	GetConnectionStatus(ctx context.Context, userID int64) (*ConnectionStatus, error)
}

type oauthService struct {
	clientID     string
	clientSecret string
	q            *store.Queries
}

func NewOAuthService(clientID, clientSecret string, q *store.Queries) OAuthService {
	return &oauthService{
		clientID:     clientID,
		clientSecret: clientSecret,
		q:            q,
	}
}

func (s *oauthService) BuildAuthorizeURL(redirectURI, state string) string {
	v := url.Values{}
	v.Set("client_id", s.clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", "repo read:org")
	v.Set("state", state)
	return authorizeURL + "?" + v.Encode()
}

type tokenAPIResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *oauthService) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	reqBody := url.Values{}
	reqBody.Set("client_id", s.clientID)
	reqBody.Set("client_secret", s.clientSecret)
	reqBody.Set("code", code)
	reqBody.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(reqBody.Encode()))
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

	var tok tokenAPIResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("github token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("github returned no access_token: %s", string(body))
	}

	res := &TokenResponse{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		Scope:        tok.Scope,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
	}
	if id, login, err := fetchUser(ctx, tok.AccessToken); err == nil {
		res.GithubUserID = id
		res.GithubLogin = login
	}
	return res, nil
}

func (s *oauthService) StoreConnection(ctx context.Context, userID int64, tokens *TokenResponse, encryptionKey string) error {
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

func (s *oauthService) GetConnectionStatus(ctx context.Context, userID int64) (*ConnectionStatus, error) {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &ConnectionStatus{Login: conn.GithubLogin}, nil
}

// GetDecryptedToken returns a valid access token, refreshing proactively if
// the token is expired or within 5 minutes of expiry.
func (s *oauthService) GetDecryptedToken(ctx context.Context, userID int64, encryptionKey string) (string, error) {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	if conn.TokenExpiresAt.Valid && time.Now().Add(5*time.Minute).After(conn.TokenExpiresAt.Time) {
		if conn.RefreshToken == nil || *conn.RefreshToken == "" {
			return "", ErrNotConnected
		}
		return s.refreshAndStore(ctx, *conn.RefreshToken, conn, userID, encryptionKey)
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

// refreshAndStore exchanges a refresh token for a new access token, persists
// the updated connection, and returns the new access token.
func (s *oauthService) refreshAndStore(ctx context.Context, refreshToken string, conn store.GithubConnection, userID int64, encryptionKey string) (string, error) {
	reqBody := url.Values{}
	reqBody.Set("client_id", s.clientID)
	reqBody.Set("client_secret", s.clientSecret)
	reqBody.Set("grant_type", "refresh_token")
	reqBody.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tok tokenAPIResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("github refresh response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("github token refresh failed: %s", string(body))
	}

	encrypted, err := encryption.Encrypt([]byte(tok.AccessToken), encryptionKey)
	if err != nil {
		return "", err
	}
	encryptedB64 := base64.StdEncoding.EncodeToString(encrypted)

	newRefreshToken := conn.RefreshToken
	if tok.RefreshToken != "" {
		newRefreshToken = &tok.RefreshToken
	}

	var expiresAt pgtype.Timestamptz
	if tok.ExpiresIn > 0 {
		expiresAt = pgtype.Timestamptz{Time: time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second), Valid: true}
	}

	_, err = s.q.UpsertGitHubConnection(ctx, store.UpsertGitHubConnectionParams{
		UserID:               userID,
		AccessTokenEncrypted: encryptedB64,
		RefreshToken:         newRefreshToken,
		TokenExpiresAt:       expiresAt,
		GithubUserID:         conn.GithubUserID,
		GithubLogin:          conn.GithubLogin,
	})
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

type userResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func fetchUser(ctx context.Context, accessToken string) (int64, string, error) {
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
	var u userResponse
	if err := json.Unmarshal(body, &u); err != nil {
		return 0, "", err
	}
	return u.ID, u.Login, nil
}
