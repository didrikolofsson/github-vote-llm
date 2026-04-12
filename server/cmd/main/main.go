package main

import (
	"context"
	"log"

	"github.com/didrikolofsson/github-vote-llm/internal/api"
	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobclient"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
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
	apiLogger := logger.New().Named("api")
	defer apiLogger.Sync()

	ctx := context.Background()
	db, err := pgxpool.New(ctx, env.DATABASE_URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	q := store.New(db)
	githubOAuthCfg := github.NewGithubOAuthConfig(
		env.GITHUB_CLIENT_ID,
		env.GITHUB_CLIENT_SECRET,
		env.SERVER_URL+"/v1/github/callback",
	)

	jobClient, err := jobclient.New(db, q, githubOAuthCfg, env)
	if err != nil {
		log.Fatalf("failed to create job client: %v", err)
	}

	s := services.New(db, q, env, githubOAuthCfg, jobClient)
	h := handlers.New(s, apiLogger, env)
	router := api.New(h, apiLogger, env)

	router.Run(":" + env.PORT)
}
