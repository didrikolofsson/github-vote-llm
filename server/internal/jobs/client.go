package jobs

import (
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type JobClientDeps struct {
	DB      *pgxpool.Pool
	Workers *river.Workers
}

func NewClient(deps JobClientDeps) (*river.Client[pgx.Tx], error) {
	client, err := river.NewClient(riverpgxv5.New(deps.DB), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: deps.Workers,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
