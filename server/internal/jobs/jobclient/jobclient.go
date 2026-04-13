package jobclient

import (
	"log/slog"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/workers"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type Client *river.Client[pgx.Tx]

func New(s *services.Services) (Client, error) {
	w := river.NewWorkers()

	cloneRepoWorker := &workers.CloneRepoWorker{
		Queries:            q,
		GithubOAuthConfig:  githubOAuthCfg,
		TokenEncryptionKey: env.TOKEN_ENCRYPTION_KEY,
	}
	runAgentWorker := &workers.RunAgentWorker{
		Queries:           q,
		GithubOAuthConfig: githubOAuthCfg,
	}

	river.AddWorker(w, cloneRepoWorker)
	river.AddWorker(w, runAgentWorker)

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: w,
	})
	if err != nil {
		return nil, err
	}

	cloneRepoWorker.RiverClient = client
	runAgentWorker.RiverClient = client

	return client, nil
}
