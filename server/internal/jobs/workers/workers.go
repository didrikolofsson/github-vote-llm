package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type RegisterWorkersDeps struct {
	Services *services.Services
	Env      *config.Environment
}

func Register(w *river.Workers, deps RegisterWorkersDeps) {
	river.AddWorker(w, &CloneRepoWorker{
		svc: deps.Services.GithubService,
	})
	river.AddWorker(w, &RunAgentWorker{
		svc: deps.Services.RunService,
	})
}
