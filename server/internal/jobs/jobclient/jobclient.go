package jobclient

import (
	"log/slog"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/workers"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func New(db *pgxpool.Pool, s *services.Services, env *config.Environment) (*river.Client[pgx.Tx], error) {
	w := river.NewWorkers()

	cloneRepoWorker, err := workers.NewCloneRepoWorker(db, s.GithubService, env.WORKSPACE_DIR)
	if err != nil {
		return nil, err
	}
	runAgentWorker := workers.NewRunAgentWorker(env.ANTHROPIC_API_KEY, db)
	openPRWorker := workers.NewOpenPRWorker(db, s.GithubService, env.WORKSPACE_DIR)

	river.AddWorker(w, cloneRepoWorker)
	river.AddWorker(w, runAgentWorker)
	river.AddWorker(w, openPRWorker)

	client, err := river.NewClient(riverpgxv5.New(db), &river.Config{
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
