package handlers

import (
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/oauth2"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Add new handlers here. Fields are exported so package api can wire routes.
type HandlerCollection struct {
	User         UserHandlers
	Auth         AuthHandlers
	Organization OrganizationHandlers
	Github       GithubHandlers
	Repository   RepositoryHandlers
	Members      MembersHandlers
	Feature      FeatureHandlers
	Portal       PortalHandlers
}

func NewHandlerCollection(conn *pgxpool.Pool, q *store.Queries, env *config.Environment, apiLogger *logger.Logger) *HandlerCollection {
	githubOAuthCfg := oauth2.NewGitHubOAuthConfig(
		env.GITHUB_CLIENT_ID,
		env.GITHUB_CLIENT_SECRET,
		env.SERVER_URL+"/v1/github/callback",
	)

	userService := services.NewUserService(conn, q)
	authService := services.NewAuthService(conn, q, env.JWT_SECRET)
	organizationService := services.NewOrganizationService(conn, q)
	githubService := services.NewGithubService(conn, q, githubOAuthCfg, env.TOKEN_ENCRYPTION_KEY)
	reposService := services.NewRepositoriesService(conn, q)
	membersService := services.NewMembersService(q)
	featuresService := services.NewFeaturesService(conn, q)
	portalService := services.NewPortalService(conn, q)

	return &HandlerCollection{
		User:         NewUserHandlers(userService, apiLogger),
		Auth:         NewAuthHandlers(authService, env.JWT_SECRET),
		Organization: NewOrganizationHandlers(organizationService, apiLogger),
		Github:       NewGithubHandlers(env, githubService),
		Repository:   NewRepositoryHandlers(reposService, apiLogger),
		Members:      NewMembersHandlers(membersService, apiLogger),
		Feature:      NewFeatureHandlers(featuresService, apiLogger),
		Portal:       NewPortalHandlers(portalService, apiLogger),
	}
}
