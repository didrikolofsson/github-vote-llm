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
	"sync"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v68/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
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

// RepoSummary is a minimal repo representation for listing.
type RepoSummary struct {
	Owner string
	Repo  string
}

type OAuthService interface {
	BuildAuthorizeURL(redirectURI, state string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)
	StoreConnection(ctx context.Context, userID int64, tokens *TokenResponse, encryptionKey string) error
	GetDecryptedToken(ctx context.Context, userID int64, encryptionKey string) (string, error)
	GetConnectionStatus(ctx context.Context, userID int64) (*ConnectionStatus, error)
	ListGitHubReposForUser(ctx context.Context, userID int64, page int, encryptionKey string) ([]RepoSummary, bool, error)
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

func (s *oauthService) oauth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		Scopes:       []string{"repo", "read:org"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authorizeURL,
			TokenURL: tokenURL,
		},
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

func tokenResponseFromOAuth(tok *oauth2.Token) *TokenResponse {
	res := &TokenResponse{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
	}
	if s, ok := tok.Extra("scope").(string); ok {
		res.Scope = s
	}
	if !tok.Expiry.IsZero() {
		res.ExpiresIn = int(time.Until(tok.Expiry).Seconds())
		if res.ExpiresIn < 0 {
			res.ExpiresIn = 0
		}
	}
	return res
}

func (s *oauthService) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	cfg := s.oauth2Config()
	cfg.RedirectURL = redirectURI
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("github oauth: empty access_token")
	}
	res := tokenResponseFromOAuth(tok)
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
// the token is expired or within 5 minutes of expiry (via oauth2).
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

	access, err := s.decryptAccess(&conn, encryptionKey)
	if err != nil {
		return "", err
	}
	return access, nil
}

func (s *oauthService) decryptAccess(conn *store.GithubConnection, encryptionKey string) (string, error) {
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

func (s *oauthService) persistOAuth2Upsert(ctx context.Context, userID int64, encryptionKey string, tok *oauth2.Token, conn *store.GithubConnection) (string, error) {
	if tok.AccessToken == "" {
		return "", fmt.Errorf("github: empty access token")
	}
	encrypted, err := encryption.Encrypt([]byte(tok.AccessToken), encryptionKey)
	if err != nil {
		return "", err
	}
	encryptedB64 := base64.StdEncoding.EncodeToString(encrypted)

	var refreshToken *string
	switch {
	case tok.RefreshToken != "":
		refreshToken = &tok.RefreshToken
	case conn != nil && conn.RefreshToken != nil:
		refreshToken = conn.RefreshToken
	}

	var expiresAt pgtype.Timestamptz
	if !tok.Expiry.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: tok.Expiry, Valid: true}
	}

	var ghUserID *int64
	var ghLogin *string
	if conn != nil {
		ghUserID = conn.GithubUserID
		ghLogin = conn.GithubLogin
	}

	_, err = s.q.UpsertGitHubConnection(ctx, store.UpsertGitHubConnectionParams{
		UserID:               userID,
		AccessTokenEncrypted: encryptedB64,
		RefreshToken:         refreshToken,
		TokenExpiresAt:       expiresAt,
		GithubUserID:         ghUserID,
		GithubLogin:          ghLogin,
	})
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (s *oauthService) refreshAndStore(ctx context.Context, refreshToken string, conn store.GithubConnection, userID int64, encryptionKey string) (string, error) {
	cfg := s.oauth2Config()
	tok, err := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return "", err
	}
	return s.persistOAuth2Upsert(ctx, userID, encryptionKey, tok, &conn)
}

// persistingTokenSource writes tokens back to the DB when oauth2 refreshes during API calls.
type persistingTokenSource struct {
	svc          *oauthService
	ctx          context.Context
	userID       int64
	key          string
	connSnapshot store.GithubConnection
	inner        oauth2.TokenSource
	lastAccess   string
	mu           sync.Mutex
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	tok, err := p.inner.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != p.lastAccess {
		if _, err := p.svc.persistOAuth2Upsert(p.ctx, p.userID, p.key, tok, &p.connSnapshot); err != nil {
			return nil, err
		}
		p.lastAccess = tok.AccessToken
	}
	return tok, nil
}

func (s *oauthService) githubClient(ctx context.Context, userID int64, encryptionKey string) (*github.Client, error) {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotConnected
		}
		return nil, err
	}

	access, err := s.decryptAccess(&conn, encryptionKey)
	if err != nil {
		return nil, err
	}

	if access == "" && (conn.RefreshToken == nil || *conn.RefreshToken == "") {
		return nil, ErrNotConnected
	}

	var expiry time.Time
	if conn.TokenExpiresAt.Valid {
		expiry = conn.TokenExpiresAt.Time
	}
	var refresh string
	if conn.RefreshToken != nil {
		refresh = *conn.RefreshToken
	}

	tok := &oauth2.Token{
		AccessToken:  access,
		RefreshToken: refresh,
		Expiry:       expiry,
	}

	cfg := s.oauth2Config()
	reuse := oauth2.ReuseTokenSource(tok, cfg.TokenSource(ctx, tok))
	wrapped := &persistingTokenSource{
		svc:          s,
		ctx:          ctx,
		userID:       userID,
		key:          encryptionKey,
		connSnapshot: conn,
		inner:        reuse,
		lastAccess:   access,
	}
	return github.NewClient(oauth2.NewClient(ctx, wrapped)), nil
}

func (s *oauthService) ListGitHubReposForUser(ctx context.Context, userID int64, page int, encryptionKey string) ([]RepoSummary, bool, error) {
	client, err := s.githubClient(ctx, userID, encryptionKey)
	if err != nil {
		return nil, false, err
	}
	return listGitHubRepos(ctx, client, page)
}

func listGitHubRepos(ctx context.Context, client *github.Client, page int) ([]RepoSummary, bool, error) {
	const perPage = 30
	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opts)
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
