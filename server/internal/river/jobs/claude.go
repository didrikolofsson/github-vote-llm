package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
)

type RunClaudeArgs struct {
	Prompt    string `json:"prompt"`
	FeatureID int64  `json:"feature_id"`
	UserID    int64  `json:"user_id"`
}

func (RunClaudeArgs) Kind() string {
	return "run_claude"
}

type RunClaudeWorker struct {
	river.WorkerDefaults[RunClaudeArgs]
}

func (w *RunClaudeWorker) Work(ctx context.Context, job *river.Job[RunClaudeArgs]) error {
	// Clone target repo to workspace
	fmt.Println("Running Claude job")
	time.Sleep(30 * time.Second)
	fmt.Println("Claude job completed")
	return nil
}
