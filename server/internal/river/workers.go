package river

import (
	"github.com/didrikolofsson/github-vote-llm/internal/river/jobs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

func NewWorkersCollection(q *store.Queries, githubOAuthCfg *oauth2.Config) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.RunClaudeWorker{
		Q:              q,
		GithubOAuthCfg: githubOAuthCfg,
	})
	return workers
}
