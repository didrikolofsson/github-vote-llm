package github

import (
	"context"

	gh "github.com/google/go-github/v84/github"
)

//go:generate mockgen -destination=mock_app_client.go -package=github github.com/didrikolofsson/github-vote-llm/internal/github AppClientIface

// AppClientIface is the interface satisfied by *AppClient, allowing test mocking.
type AppClientIface interface {
	InstallationClient(ctx context.Context, installationID int64) (*gh.Client, error)
	InstallationToken(ctx context.Context, installationID int64) (string, error)
	AppAPIClient() (*gh.Client, error)
}
