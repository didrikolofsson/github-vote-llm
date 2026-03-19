package github

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/jferrl/go-githubauth"
	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// AppConfig holds GitHub App authentication configuration.
type AppConfig struct {
	AppID           int64
	PrivateKeyBytes []byte // PEM bytes from GITHUB_PRIVATE_KEY env var
	WebhookSecret   string
}

// ClientFactory creates per-installation GitHub clients for a GitHub App.
type ClientFactory struct {
	appTokenSource oauth2.TokenSource
	log            *logger.Logger
}

// NewClientFactory creates a ClientFactory from a GitHub App private key.
func NewClientFactory(cfg AppConfig, log *logger.Logger) (*ClientFactory, error) {
	if len(cfg.PrivateKeyBytes) == 0 {
		return nil, fmt.Errorf("private key is required (set GITHUB_PRIVATE_KEY)")
	}

	appTokenSource, err := githubauth.NewApplicationTokenSource(cfg.AppID, cfg.PrivateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("create app token source: %w", err)
	}

	return &ClientFactory{
		appTokenSource: appTokenSource,
		log:            log.Named("github"),
	}, nil
}

// clientForInstallation creates an authenticated gh.Client and token source
// for a specific GitHub App installation.
func (f *ClientFactory) clientForInstallation(installationID int64) (*gh.Client, oauth2.TokenSource, error) {
	tokenSource := githubauth.NewInstallationTokenSource(installationID, f.appTokenSource)
	httpClient := oauth2.NewClient(context.Background(), tokenSource)
	ghClient := gh.NewClient(httpClient)
	return ghClient, tokenSource, nil
}
