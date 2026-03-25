package main

import (
	"context"
	"log"

	"github.com/didrikolofsson/github-vote-llm/internal/api"
	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	appoauth2 "github.com/didrikolofsson/github-vote-llm/internal/oauth2"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if gin.Mode() == gin.DebugMode {
		if err := godotenv.Load(); err != nil {
			log.Fatalf("failed to load .env file: %v", err)
		}
	}

	env, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}

	appLogger := logger.New().Named("main")
	defer appLogger.Sync()

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, env.DATABASE_URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	apiLogger := logger.New().Named("api")
	defer apiLogger.Sync()

	q := store.New(conn)
	userService := services.NewUserService(conn, q)
	userHandlers := handlers.NewUserHandlers(userService, apiLogger)
	authService := services.NewAuthService(conn, q, env.JWT_SECRET)
	authHandlers := handlers.NewAuthHandlers(authService, env.JWT_SECRET)
	organizationService := services.NewOrganizationService(conn, q)
	organizationHandlers := handlers.NewOrganizationHandlers(organizationService, apiLogger)
	reposService := services.NewRepositoriesService(conn, q)
	reposHandlers := handlers.NewRepositoryHandlers(reposService, apiLogger)
	membersService := services.NewMembersService(q)
	membersHandlers := handlers.NewMembersHandlers(membersService, apiLogger)

	githubOAuthCfg := appoauth2.NewGitHubOAuthConfig(
		env.GITHUB_CLIENT_ID,
		env.GITHUB_CLIENT_SECRET,
		env.SERVER_URL+"/v1/github/callback",
	)
	githubService := services.NewGithubService(conn, q, githubOAuthCfg, env.TOKEN_ENCRYPTION_KEY)
	githubHandlers := handlers.NewGithubHandlers(env, githubService)

	router := api.NewRestApiRouter(
		env,
		apiLogger,
		userHandlers,
		authHandlers,
		organizationHandlers,
		githubHandlers,
		reposHandlers,
		membersHandlers,
	).Create()

	router.Run(":" + env.PORT)
}
