package jobclient

import (
	"log/slog"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/workers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"golang.org/x/oauth2"
)

func New(
	pool *pgxpool.Pool,
	q *store.Queries,
	githubOAuthCfg *oauth2.Config,
	env *config.Environment,
) (*river.Client[pgx.Tx], error) {
	w := river.NewWorkers()

	river.AddWorker(w, &workers.CloneRepoWorker{
		Queries:            q,
		Config:             githubOAuthCfg,
		TokenEncryptionKey: env.TOKEN_ENCRYPTION_KEY,
	})

	river.AddWorker(w, &workers.RunAgentWorker{
		Queries:           q,
		GithubOAuthConfig: githubOAuthCfg,
	})

	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: w,
	})
}
