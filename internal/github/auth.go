package github

import (
	"context"
	"fmt"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/jferrl/go-githubauth"
	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// AppConfig holds GitHub App authentication configuration.
type AppConfig struct {
	AppID          int64
	PrivateKeyPath string
	WebhookSecret  string
}

// ClientFactory creates per-installation GitHub clients for a GitHub App.
type ClientFactory struct {
	appTokenSource oauth2.TokenSource
	log            *logger.Logger
}

// NewClientFactory creates a ClientFactory from a GitHub App private key.
func NewClientFactory(cfg AppConfig, log *logger.Logger) (*ClientFactory, error) {
	key, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", cfg.PrivateKeyPath, err)
	}

	appTokenSource, err := githubauth.NewApplicationTokenSource(cfg.AppID, key)
	if err != nil {
		return nil, fmt.Errorf("create app token source: %w", err)
	}

	return &ClientFactory{
		appTokenSource: appTokenSource,
		log:            log.Named("github-app"),
	}, nil
}

// ClientForInstallation creates a ClientAPI for a specific GitHub App installation.
func (f *ClientFactory) ClientForInstallation(installationID int64) (ClientAPI, error) {
	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, f.appTokenSource)
	httpClient := oauth2.NewClient(context.Background(), installationTokenSource)
	ghClient := gh.NewClient(httpClient)

	return &AppClient{
		gh:          ghClient,
		tokenSource: installationTokenSource,
		log:         f.log,
	}, nil
}

// AppClient implements ClientAPI using GitHub App installation tokens.
type AppClient struct {
	gh          *gh.Client
	tokenSource oauth2.TokenSource
	log         *logger.Logger
}

// Compile-time check that AppClient implements ClientAPI.
var _ ClientAPI = (*AppClient)(nil)

func (c *AppClient) GetInstallationToken(ctx context.Context) (string, error) {
	token, err := c.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get installation token: %w", err)
	}
	return token.AccessToken, nil
}

func (c *AppClient) GetIssue(ctx context.Context, owner, repo string, number int) (*gh.Issue, error) {
	issue, _, err := c.gh.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("get issue #%d: %w", number, err)
	}
	return issue, nil
}

func (c *AppClient) GetReactionCount(ctx context.Context, owner, repo string, issueNumber int) (int, error) {
	var total int
	opts := &gh.ListOptions{PerPage: 100}
	for {
		reactions, resp, err := c.gh.Reactions.ListIssueReactions(ctx, owner, repo, issueNumber, opts)
		if err != nil {
			return 0, fmt.Errorf("list reactions for issue #%d: %w", issueNumber, err)
		}
		for _, r := range reactions {
			if r.GetContent() == "+1" {
				total++
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return total, nil
}

func (c *AppClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, _, err := c.gh.Issues.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{label})
	if err != nil {
		return fmt.Errorf("add label %q to issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

func (c *AppClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, err := c.gh.Issues.RemoveLabelForIssue(ctx, owner, repo, issueNumber, label)
	if err != nil {
		return fmt.Errorf("remove label %q from issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

func (c *AppClient) CreateComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	comment := &gh.IssueComment{Body: gh.Ptr(body)}
	_, _, err := c.gh.Issues.CreateComment(ctx, owner, repo, issueNumber, comment)
	if err != nil {
		return fmt.Errorf("create comment on issue #%d: %w", issueNumber, err)
	}
	return nil
}

func (c *AppClient) CreatePullRequest(ctx context.Context, owner, repo, head, base, title, body string) (*gh.PullRequest, error) {
	pr := &gh.NewPullRequest{
		Title: gh.Ptr(title),
		Body:  gh.Ptr(body),
		Head:  gh.Ptr(head),
		Base:  gh.Ptr(base),
	}
	created, _, err := c.gh.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	return created, nil
}

func (c *AppClient) FindPullRequestByHead(ctx context.Context, owner, repo, head string) (*gh.PullRequest, error) {
	prs, _, err := c.gh.PullRequests.List(ctx, owner, repo, &gh.PullRequestListOptions{
		Head:  owner + ":" + head,
		State: "open",
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for head %s: %w", head, err)
	}
	if len(prs) > 0 {
		return prs[0], nil
	}
	return nil, nil
}

func (c *AppClient) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	r, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("get repo %s/%s: %w", owner, repo, err)
	}
	return r.GetDefaultBranch(), nil
}
