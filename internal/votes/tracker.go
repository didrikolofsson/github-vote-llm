package votes

import (
	"context"
	"fmt"
	"log"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
)

// Tracker monitors vote counts on issues and applies threshold labels.
type Tracker struct {
	client *gh.Client
}

// NewTracker creates a new vote tracker.
func NewTracker(client *gh.Client) *Tracker {
	return &Tracker{client: client}
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

	log.Printf("votes: issue #%d in %s/%s has %d votes (threshold: %d)", issueNumber, owner, repo, count, repoCfg.VoteThreshold)

	if count >= repoCfg.VoteThreshold {
		label := fmt.Sprintf("votes:%d+", repoCfg.VoteThreshold)
		if err := t.client.AddLabel(ctx, owner, repo, issueNumber, label); err != nil {
			return fmt.Errorf("adding threshold label: %w", err)
		}
		log.Printf("votes: labeled issue #%d with %q", issueNumber, label)
	}

	return nil
}
