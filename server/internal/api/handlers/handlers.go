package handlers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobclient"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

type Handlers struct {
	User         UserHandlers
	Auth         AuthHandlers
	Organization OrganizationHandlers
	Github       GithubHandlers
	Repository   RepositoryHandlers
	Runs         RunsHandlers
	Members      MembersHandlers
	Feature      FeatureHandlers
	Portal       PortalHandlers
}

func New(
	s *services.Services,
	jc jobclient.Client,
	logger *logger.Logger,
	env *config.Environment,
) Handlers {
	return Handlers{
		User:         NewUserHandlers(s.UserService, logger),
		Auth:         NewAuthHandlers(s.AuthService, env.JWT_SECRET),
		Organization: NewOrganizationHandlers(s.OrganizationService, logger),
		Github:       NewGithubHandlers(env, s.GithubService),
		Repository:   NewRepositoryHandlers(s.RepositoriesService, logger),
		Runs:         NewRunsHandlers(s.RunService, env),
		Members:      NewMembersHandlers(s.MembersService, logger),
		Feature:      NewFeatureHandlers(s.FeaturesService, logger),
		Portal:       NewPortalHandlers(s.PortalService, logger, hub.NewHub()),
	}
}
