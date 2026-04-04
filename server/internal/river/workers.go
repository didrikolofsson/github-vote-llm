package river

import (
	"github.com/didrikolofsson/github-vote-llm/internal/river/jobs"
	"github.com/riverqueue/river"
)

func NewWorkersCollection() *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.RunClaudeWorker{})
	return workers
}
