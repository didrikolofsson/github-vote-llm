package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v68/github"
)

// Client wraps the go-github client with methods needed by vote-llm.
type Client struct {
	gh *gh.Client
}

// NewClient creates a Client authenticated with the given token.
func NewClient(token string) *Client {
	return &Client{
		gh: gh.NewClient(nil).WithAuthToken(token),
	}
}

// GetIssue fetches an issue by number.
func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*gh.Issue, error) {
	issue, _, err := c.gh.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("get issue #%d: %w", number, err)
	}
	return issue, nil
}

// GetReactionCount returns the number of +1 (thumbs up) reactions on an issue.
func (c *Client) GetReactionCount(ctx context.Context, owner, repo string, issueNumber int) (int, error) {
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

// AddLabel adds a label to an issue.
func (c *Client) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, _, err := c.gh.Issues.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{label})
	if err != nil {
		return fmt.Errorf("add label %q to issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

// RemoveLabel removes a label from an issue.
func (c *Client) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, err := c.gh.Issues.RemoveLabelForIssue(ctx, owner, repo, issueNumber, label)
	if err != nil {
		return fmt.Errorf("remove label %q from issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

// CreateComment posts a comment on an issue.
func (c *Client) CreateComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	comment := &gh.IssueComment{Body: gh.Ptr(body)}
	_, _, err := c.gh.Issues.CreateComment(ctx, owner, repo, issueNumber, comment)
	if err != nil {
		return fmt.Errorf("create comment on issue #%d: %w", issueNumber, err)
	}
	return nil
}

// CreatePullRequest opens a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, head, base, title, body string) (*gh.PullRequest, error) {
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

// GetDefaultBranch returns the default branch name for a repository.
func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	r, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("get repo %s/%s: %w", owner, repo, err)
	}
	return r.GetDefaultBranch(), nil
}
