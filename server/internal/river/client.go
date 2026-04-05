package river

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"golang.org/x/oauth2"
)

func NewRiverClient(ctx context.Context, pool *pgxpool.Pool, q *store.Queries, githubOAuthCfg *oauth2.Config) *river.Client[pgx.Tx] {
	workers := NewWorkersCollection(q, githubOAuthCfg)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("failed to create river client: %v", err)
	}
	return riverClient
}
