package handlers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
)

type Handlers struct {
	User         *UserHandlers
	Auth         *AuthHandlers
	Organization *OrganizationHandlers
	Github       *GithubHandlers
	Webhooks     *WebhooksHandlers
	Repository   *RepositoryHandlers
	Runs         *RunsHandlers
	Members      *MembersHandlers
	Feature      *FeatureHandlers
	Portal       *PortalHandlers
}

type NewHandlersDeps struct {
	Services *services.Services
	Logger   *logger.Logger
	Hub      hub.Hub
	Env      *config.Environment
}

func New(
	deps NewHandlersDeps,
) Handlers {
	return Handlers{
		User:         NewUserHandlers(deps.Services.UserService, deps.Logger),
		Auth:         NewAuthHandlers(deps.Services.AuthService),
		Organization: NewOrganizationHandlers(deps.Services.OrganizationService, deps.Logger),
		Github:       NewGithubHandlers(deps.Services.GithubService, deps.Env.FRONTEND_URL),
		Webhooks:     NewWebhooksHandlers(deps.Services.GithubService, deps.Env.GITHUB_APP_WEBHOOK_SECRET, deps.Logger),
		Repository:   NewRepositoryHandlers(deps.Services.RepositoriesService, deps.Logger),
		Runs:         NewRunsHandlers(deps.Services.RunService, deps.Logger),
		Members:      NewMembersHandlers(deps.Services.MembersService, deps.Logger),
		Feature:      NewFeatureHandlers(deps.Services.FeaturesService, deps.Logger),
		Portal:       NewPortalHandlers(deps.Services.PortalService, deps.Logger, deps.Hub),
	}
}
