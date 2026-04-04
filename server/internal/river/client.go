package river

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func NewRiverClient(ctx context.Context, pool *pgxpool.Pool) *river.Client[pgx.Tx] {
	workers := NewWorkersCollection()
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
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
