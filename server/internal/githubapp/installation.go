package githubapp

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"golang.org/x/oauth2"
)

var ErrInstallationNotFound = errors.New("github: installation not found")

// Client authenticates as the GitHub App and mints installation access tokens.
// Tokens are cached in-memory per installation until ~1 minute before expiry.
type Client struct {
	appID      int64
	privateKey *rsa.PrivateKey
	httpClient *http.Client
	now        func() time.Time

	mu    sync.Mutex
	cache map[int64]*cachedToken
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewClient(appID int64, privateKey *rsa.PrivateKey) *Client {
	return &Client{
		appID:      appID,
		privateKey: privateKey,
		httpClient: http.DefaultClient,
		now:        time.Now,
		cache:      make(map[int64]*cachedToken),
	}
}

// AppJWT returns a fresh app-level JWT suitable for authenticating GitHub App
// endpoints like GET /app/installations/{id}.
func (c *Client) AppJWT() (string, error) {
	return GenerateAppJWT(c.appID, c.privateKey, c.now())
}

// AppHTTPClient returns an http.Client whose requests carry a fresh app JWT.
// Use for endpoints authenticated as the app (not as an installation).
func (c *Client) AppHTTPClient(ctx context.Context) *http.Client {
	return &http.Client{Transport: &appTransport{c: c, base: http.DefaultTransport}}
}

type appTransport struct {
	c    *Client
	base http.RoundTripper
}

func (t *appTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	j, err := t.c.AppJWT()
	if err != nil {
		return nil, err
	}
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+j)
	r.Header.Set("Accept", "application/vnd.github+json")
	return t.base.RoundTrip(r)
}

// AppGithubClient returns a go-github client authenticated as the app.
func (c *Client) AppGithubClient(ctx context.Context) *github.Client {
	return github.NewClient(c.AppHTTPClient(ctx))
}

// InstallationToken returns a cached-or-fresh installation access token.
func (c *Client) InstallationToken(ctx context.Context, installationID int64) (string, time.Time, error) {
	c.mu.Lock()
	if t, ok := c.cache[installationID]; ok && c.now().Before(t.expiresAt.Add(-60*time.Second)) {
		defer c.mu.Unlock()
		return t.token, t.expiresAt, nil
	}
	c.mu.Unlock()

	tok, exp, err := c.mintInstallationToken(ctx, installationID)
	if err != nil {
		return "", time.Time{}, err
	}

	c.mu.Lock()
	c.cache[installationID] = &cachedToken{token: tok, expiresAt: exp}
	c.mu.Unlock()

	return tok, exp, nil
}

func (c *Client) mintInstallationToken(ctx context.Context, installationID int64) (string, time.Time, error) {
	appJWT, err := c.AppJWT()
	if err != nil {
		return "", time.Time{}, err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("githubapp: mint installation token: http %d", resp.StatusCode)
	}

	var parsed installationTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", time.Time{}, fmt.Errorf("githubapp: decode installation token: %w", err)
	}
	return parsed.Token, parsed.ExpiresAt, nil
}

// GetInstallation fetches metadata about an installation (GET /app/installations/{id}).
func (c *Client) GetInstallation(ctx context.Context, installationID int64) (*github.Installation, error) {
	inst, _, err := c.AppGithubClient(ctx).Apps.GetInstallation(ctx, installationID)
	return inst, err
}

// InstallationTokenSource returns an oauth2.TokenSource for a specific installation.
// Use with oauth2.NewClient to build an HTTP client, or pass to go-github.
func (c *Client) InstallationTokenSource(ctx context.Context, installationID int64) oauth2.TokenSource {
	return &installationTokenSource{c: c, ctx: ctx, installationID: installationID}
}

// InstallationGithubClient returns a go-github client authenticated as the given installation.
func (c *Client) InstallationGithubClient(ctx context.Context, installationID int64) *github.Client {
	ts := c.InstallationTokenSource(ctx, installationID)
	return github.NewClient(oauth2.NewClient(ctx, ts))
}

type installationTokenSource struct {
	c              *Client
	ctx            context.Context
	installationID int64
}

func (s *installationTokenSource) Token() (*oauth2.Token, error) {
	tok, exp, err := s.c.InstallationToken(s.ctx, s.installationID)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{AccessToken: tok, TokenType: "token", Expiry: exp}, nil
}
