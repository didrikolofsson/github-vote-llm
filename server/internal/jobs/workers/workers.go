package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/riverqueue/river"
)

type RegisterWorkersDeps struct {
	Services *services.Services
	Env      *config.Environment
	Logger   *logger.Logger
}

func Register(w *river.Workers, deps RegisterWorkersDeps) {
	river.AddWorker(w, &CloneRepoWorker{
		svc: deps.Services.GithubService,
	})
	river.AddWorker(w, &RunAgentWorker{
		svc: deps.Services.RunService,
	})
	river.AddWorker(w, &OpenPRWorker{
		svc: deps.Services.GithubService,
		log: deps.Logger.Named("open-pr"),
	})
}
