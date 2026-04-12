package workers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"

	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

type CloneRepoWorker struct {
	river.WorkerDefaults[jobargs.CloneRepoArgs]
	Queries            *store.Queries
	Config             *oauth2.Config
	TokenEncryptionKey string
}

var (
	ErrInvalidCloneURL        = errors.New("github: invalid or missing clone URL")
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

func (w *CloneRepoWorker) Work(
	ctx context.Context, job *river.Job[jobargs.CloneRepoArgs],
) error {
	conn, err := w.Queries.GetGitHubConnectionByUserID(ctx, job.Args.UserID)
	if err != nil {
		return err
	}

	if conn.AccessTokenEncrypted == "" {
		return ErrGitHubNotConnected
	}

	ts := github.NewGithubTokenSource(
		ctx,
		w.Queries,
		w.Config,
		conn.UserID,
		w.TokenEncryptionKey,
	)

	tok, err := ts.Token()
	if err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return ErrGitHubTokenUnavailable
	}

	client := github.NewGithubClientByUserID(
		ctx,
		w.Queries,
		w.Config,
		conn.UserID,
		w.TokenEncryptionKey,
	)

	repo, _, err := client.Repositories.Get(ctx, job.Args.Owner, job.Args.Name)
	if err != nil {
		return err
	}
	if repo.CloneURL == nil || *repo.CloneURL == "" {
		return ErrInvalidCloneURL
	}
	cloneURL := *repo.CloneURL

	u, err := url.Parse(cloneURL)
	if err != nil {
		return fmt.Errorf("parse clone url: %w", err)
	}
	u.User = url.UserPassword("x-access-token", tok.AccessToken)
	authCloneURL := u.String()

	cmd := exec.CommandContext(ctx, "git", "clone", authCloneURL)
	cmd.Dir = job.Args.Workspace
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
