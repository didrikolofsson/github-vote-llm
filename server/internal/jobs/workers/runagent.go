package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type RunAgentWorker struct {
	river.WorkerDefaults[args.RunAgentArgs]
	ApiKey      string
	db          *pgxpool.Pool
	RiverClient *river.Client[pgx.Tx]
}

// func NewRunAgentWorker(apiKey string, db *pgxpool.Pool) *RunAgentWorker {
// 	return &RunAgentWorker{ApiKey: apiKey, db: db}
// }

// func (w *RunAgentWorker) Work(ctx context.Context, job *river.Job[args.RunAgentArgs]) error {
// 	runner := claude.NewClaudeRunner(claude.NewClaudeRunnerParams{
// 		ApiKey:  w.ApiKey,
// 		WorkDir: job.Args.WorkDir,
// 	})

// 	ch, err := runner.Run(ctx, job.Args.Prompt)
// 	if err != nil {
// 		return w.fail(ctx, job.Args.RunID, err)
// 	}

// 	for event := range ch {
// 		if event.Err != nil {
// 			return w.fail(ctx, job.Args.RunID, event.Err)
// 		}
// 	}

// 	tx, err := w.db.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(ctx)

// 	if _, err := w.RiverClient.InsertTx(ctx, tx, args.OpenPRArgs{
// 		UserID:     job.Args.UserID,
// 		RunID:      job.Args.RunID,
// 		Owner:      job.Args.Owner,
// 		Name:       job.Args.Name,
// 		BranchName: job.Args.BranchName,
// 		Prompt:     job.Args.Prompt,
// 	}, nil); err != nil {
// 		return err
// 	}

// 	return tx.Commit(ctx)
// }

// // fail marks the run as failed and returns the original error so River records it.
// func (w *RunAgentWorker) fail(ctx context.Context, runID int64, cause error) error {
// 	q := store.New(w.db)
// 	_ = q.UpdateRunStatus(ctx, store.UpdateRunStatusParams{
// 		Status: store.FeatureRunStatusFailed,
// 		ID:     runID,
// 	})
// 	return fmt.Errorf("agent: %w", cause)
// }
