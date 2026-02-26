package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if gin.Mode() == gin.DebugMode {
		if err := godotenv.Load(".env.development"); err != nil {
			log.Fatalf("failed to load .env file: %v", err)
		}
	}

	key := os.Getenv("GITHUB_PRIVATE_KEY")
	if key == "" {
		log.Fatal("GITHUB_PRIVATE_KEY is required")
	}

	workspaceDir := os.Getenv("WORKSPACE_DIR")
	if workspaceDir == "" {
		workspaceDir = config.DefaultWorkspaceDir
	}

	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		log.Fatal("GITHUB_APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		log.Fatalf("GITHUB_APP_ID must be a number: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	privateKeyBytes := []byte(key)

	appLog := logger.New()
	defer appLog.Sync()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	st := store.NewPostgresStore(pool)

	factory, err := ghclient.NewClientFactory(ghclient.AppConfig{
		AppID:           appID,
		PrivateKeyBytes: privateKeyBytes,
	}, appLog)
	if err != nil {
		log.Fatalf("failed to create GitHub App client factory: %v", err)
	}

	webhookHandler := handlers.NewWebhookHandler(factory, appLog, workspaceDir, st)

	router := gin.New()
	router.SetTrustedProxies(nil)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	webhooks := router.Group("/github")
	webhooks.Use(middleware.ValidateSignature())
	webhooks.POST("/webhook", webhookHandler.HandleGithubWebhook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
