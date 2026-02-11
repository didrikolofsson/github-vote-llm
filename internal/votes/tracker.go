package votes

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

// Tracker monitors vote counts on issues and applies threshold labels.
type Tracker struct {
	client *gh.Client
	log    *logger.Logger
}

// NewTracker creates a new vote tracker.
func NewTracker(client *gh.Client, log *logger.Logger) *Tracker {
	return &Tracker{client: client, log: log.Named("votes")}
}

// CountVotes returns the number of +1 reactions on an issue.
func (t *Tracker) CountVotes(ctx context.Context, owner, repo string, issueNumber int) (int, error) {
	return t.client.GetReactionCount(ctx, owner, repo, issueNumber)
}

// CheckAndLabel checks if an issue has reached the vote threshold and labels it accordingly.
func (t *Tracker) CheckAndLabel(ctx context.Context, owner, repo string, issueNumber int, repoCfg *config.RepoConfig) error {
	count, err := t.CountVotes(ctx, owner, repo, issueNumber)
	if err != nil {
		return fmt.Errorf("counting votes: %w", err)
	}

	t.log.Infow("vote count", "issue", issueNumber, "repo", owner+"/"+repo, "votes", count, "threshold", repoCfg.VoteThreshold)

	if count >= repoCfg.VoteThreshold {
		label := fmt.Sprintf("votes:%d+", repoCfg.VoteThreshold)
		if err := t.client.AddLabel(ctx, owner, repo, issueNumber, label); err != nil {
			return fmt.Errorf("adding threshold label: %w", err)
		}
		t.log.Infow("labeled issue", "issue", issueNumber, "label", label)
	}

	return nil
}
