package votes

import (
	"context"
	"fmt"

	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

// Tracker monitors vote counts on issues and applies threshold labels.
type Tracker struct {
	client gh.ClientAPI
	log    *logger.Logger
}

// NewTracker creates a new vote tracker.
func NewTracker(client gh.ClientAPI, log *logger.Logger) *Tracker {
	return &Tracker{client: client, log: log.Named("votes")}
}

// CountVotes returns the number of +1 reactions on an issue.
func (t *Tracker) CountVotes(ctx context.Context, owner, repo string, issueNumber int) (int, error) {
	return t.client.GetReactionCount(ctx, owner, repo, issueNumber)
}

// CheckAndLabel checks if an issue has reached the vote threshold and labels it.
// Returns true if the threshold was met.
func (t *Tracker) CheckAndLabel(ctx context.Context, owner, repo string, issueNumber int, repoCfg *config.RepoConfig) (bool, error) {
	count, err := t.CountVotes(ctx, owner, repo, issueNumber)
	if err != nil {
		return false, fmt.Errorf("counting votes: %w", err)
	}

	t.log.Infow("vote count", "issue", issueNumber, "repo", owner+"/"+repo, "votes", count, "threshold", repoCfg.VoteThreshold)

	if count >= repoCfg.VoteThreshold {
		if err := t.client.AddLabel(ctx, owner, repo, issueNumber, repoCfg.Labels.Candidate); err != nil {
			return false, fmt.Errorf("adding candidate label: %w", err)
		}
		t.log.Infow("labeled issue as candidate", "issue", issueNumber, "label", repoCfg.Labels.Candidate)
		return true, nil
	}

	return false, nil
}
