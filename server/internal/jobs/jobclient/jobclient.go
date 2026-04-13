package jobclient

import (
	"log/slog"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/workers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func New(pool *pgxpool.Pool, githubSvc services.GithubService, env *config.Environment) (*river.Client[pgx.Tx], error) {
	w := river.NewWorkers()

	cloneRepoWorker, err := workers.NewCloneRepoWorker(pool, githubSvc, env.WORKSPACE_DIR)
	if err != nil {
		return nil, err
	}
	runAgentWorker := workers.NewRunAgentWorker(env.ANTHROPIC_API_KEY)

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
