package main

import (
	"context"
	"log"
	"net/http"
	"os"

	apimw "github.com/didrikolofsson/github-vote-llm/internal/api"
	apihandlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/didrikolofsson/github-vote-llm/internal/webhook"
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

	appLog := logger.New()
	defer appLog.Sync()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, env.DATABASE_URL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	router := gin.New()
	router.SetTrustedProxies(nil)

	api := router.Group("/v1/api")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	st := store.NewPostgresStore(pool)
	factory, err := ghclient.NewClientFactory(ghclient.AppConfig{
		AppID:           env.GITHUB_APP_ID,
		PrivateKeyBytes: []byte(env.GITHUB_PRIVATE_KEY),
	}, appLog)
	if err != nil {
		log.Fatalf("failed to create GitHub App client factory: %v", err)
	}

	webhookHandler := webhook.NewWebhookHandler(factory, appLog, env.WORKSPACE_DIR, st)

	webhooks := api.Group("/github")
	webhooks.Use(webhook.ValidateSignature(env.WEBHOOK_SECRET))
	webhooks.POST("/webhook", webhookHandler.HandleGithubWebhook)

	runsService := services.NewRunsService(st)
	reposService := services.NewReposService(st)
	runsHandler := apihandlers.NewRunsHandler(runsService)
	reposHandler := apihandlers.NewReposHandler(reposService)

	api.Use(apimw.ValidateAPIKey(env.API_KEY))

	api.GET("/runs", runsHandler.List)
	api.POST("/runs", runsHandler.Create)
	api.GET("/runs/:id", runsHandler.Get)
	api.POST("/runs/:id/retry", runsHandler.Retry)
	api.POST("/runs/:id/cancel", runsHandler.Cancel)
	api.GET("/repos", reposHandler.List)
	api.GET("/repos/:owner/:repo/config", reposHandler.GetConfig)
	api.PUT("/repos/:owner/:repo/config", reposHandler.UpdateConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
