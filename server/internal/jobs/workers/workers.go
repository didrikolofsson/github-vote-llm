package workers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type RegisterWorkersDeps struct {
	Services  *services.Services
	Env       *config.Environment
	Logger    *logger.Logger
	JobClient *river.Client[pgx.Tx]
}

func Register(w *river.Workers, deps RegisterWorkersDeps) {
	river.AddWorker(w, &CloneRepoWorker{
		svc:    deps.Services.GithubService,
		runSvc: deps.Services.RunService,
		jc:     deps.JobClient,
	})
	river.AddWorker(w, &RunAgentWorker{
		svc: deps.Services.RunService,
	})
	river.AddWorker(w, &OpenPRWorker{
		svc:    deps.Services.GithubService,
		runSvc: deps.Services.RunService,
		log:    deps.Logger.Named("open-pr"),
	})
}
