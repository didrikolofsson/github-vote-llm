package handlers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

// Add new handlers here. Fields are exported so package api can wire routes.
type HandlerCollection struct {
	User         UserHandlers
	Auth         AuthHandlers
	Organization OrganizationHandlers
	Github       GithubHandlers
	Repository   RepositoryHandlers
	Run          RunsHandlers
	Members      MembersHandlers
	Feature      FeatureHandlers
	Portal       PortalHandlers
}

type NewHandlerCollectionParams struct {
	Conn           *pgxpool.Pool
	Queries        *store.Queries
	Env            *config.Environment
	ApiLogger      *logger.Logger
	RiverClient    *river.Client[pgx.Tx]
	GithubOAuthCfg *oauth2.Config
}

func NewHandlerCollection(p NewHandlerCollectionParams) *HandlerCollection {
	userService := services.NewUserService(p.Conn, p.Queries)
	authService := services.NewAuthService(p.Conn, p.Queries, p.Env.JWT_SECRET)
	organizationService := services.NewOrganizationService(p.Conn, p.Queries)
	githubService := services.NewGithubService(p.Conn, p.Queries, &services.GithubServiceConfigParams{
		TokenEncryptionKey: p.Env.TOKEN_ENCRYPTION_KEY,
		Config:             *p.GithubOAuthCfg,
	})
	reposService := services.NewRepositoriesService(p.Conn, p.Queries)
	membersService := services.NewMembersService(p.Queries)
	runService := services.NewRunService(p.Conn, p.Queries, p.RiverClient)
	portalEventHub := hub.NewHub()
	featuresService := services.NewFeaturesService(p.Conn, p.Queries, portalEventHub)
	portalService := services.NewPortalService(p.Conn, p.Queries)

	return &HandlerCollection{
		User:         NewUserHandlers(userService, p.ApiLogger),
		Auth:         NewAuthHandlers(authService, p.Env.JWT_SECRET),
		Organization: NewOrganizationHandlers(organizationService, p.ApiLogger),
		Github:       NewGithubHandlers(p.Env, githubService),
		Repository:   NewRepositoryHandlers(reposService, p.ApiLogger),
		Run:          NewRunsHandlers(runService),
		Members:      NewMembersHandlers(membersService, p.ApiLogger),
		Feature:      NewFeatureHandlers(featuresService, p.ApiLogger),
		Portal:       NewPortalHandlers(portalService, p.ApiLogger, portalEventHub),
	}
}
