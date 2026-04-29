package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gh "github.com/google/go-github/v84/github"
)

// AppClient authenticates as a GitHub App and creates installation-scoped clients.
type AppClient struct {
	appID      int64
	privateKey []byte
}

// NewAppClient parses GITHUB_APP_ID and GITHUB_APP_PRIVATE_KEY from the environment.
// privateKeyRaw may be a PEM string or base64-encoded PEM.
func NewAppClient(appIDStr, privateKeyRaw string) (*AppClient, error) {
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse GITHUB_APP_ID: %w", err)
	}

	key := []byte(privateKeyRaw)

	// Accept base64-encoded PEM (useful when storing in env vars).
	if !strings.HasPrefix(strings.TrimSpace(privateKeyRaw), "-----") {
		decoded, err := base64.StdEncoding.DecodeString(privateKeyRaw)
		if err != nil {
			return nil, fmt.Errorf("decode GITHUB_APP_PRIVATE_KEY: %w", err)
		}
		key = decoded
	}

	return &AppClient{appID: appID, privateKey: key}, nil
}

// InstallationClient returns a go-github client authenticated for the given installation.
// Use this for GitHub API calls (create PR, manage labels, etc.).
func (c *AppClient) InstallationClient(ctx context.Context, installationID int64) (*gh.Client, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, c.appID, installationID, c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create installation transport: %w", err)
	}
	return gh.NewClient(&http.Client{Transport: itr}), nil
}

// InstallationToken returns a short-lived token string for the given installation.
// Use this for authenticated git operations: https://x-access-token:{token}@github.com/...
func (c *AppClient) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, c.appID, installationID, c.privateKey)
	if err != nil {
		return "", fmt.Errorf("create installation transport: %w", err)
	}
	token, err := itr.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("get installation token: %w", err)
	}
	return token, nil
}

// AppAPIClient returns a go-github client authenticated as the App itself (not as an installation).
// Use this for App-level API calls, e.g. verifying installation status.
func (c *AppClient) AppAPIClient() (*gh.Client, error) {
	tr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, c.appID, c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create app transport: %w", err)
	}
	return gh.NewClient(&http.Client{Transport: tr}), nil
}
