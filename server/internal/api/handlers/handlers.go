package handlers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
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
	jc *river.Client[pgx.Tx],
	logger *logger.Logger,
) Handlers {
	return Handlers{
		User:         NewUserHandlers(s.UserService, logger),
		Auth:         NewAuthHandlers(s.AuthService),
		Organization: NewOrganizationHandlers(s.OrganizationService, logger),
		Github:       NewGithubHandlers(s.GithubService),
		Repository:   NewRepositoryHandlers(s.RepositoriesService, logger),
		Runs:         NewRunsHandlers(s.RunService, jc),
		Members:      NewMembersHandlers(s.MembersService, logger),
		Feature:      NewFeatureHandlers(s.FeaturesService, logger),
		Portal:       NewPortalHandlers(s.PortalService, logger, hub.NewHub()),
	}
}
