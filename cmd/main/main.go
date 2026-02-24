package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if gin.Mode() == gin.DebugMode {
		if err := godotenv.Load(".env.development"); err != nil {
			log.Fatalf("failed to load .env file: %v", err)
		}
	}

	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		log.Fatal("GITHUB_APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		log.Fatalf("GITHUB_APP_ID must be a number: %v", err)
	}

	key := os.Getenv("GITHUB_PRIVATE_KEY")
	if key == "" {
		log.Fatal("GITHUB_PRIVATE_KEY is required")
	}

	privateKeyBytes := []byte(key)

	appLog := logger.New()
	defer appLog.Sync()

	factory, err := ghclient.NewClientFactory(ghclient.AppConfig{
		AppID:           appID,
		PrivateKeyBytes: privateKeyBytes,
	}, appLog)
	if err != nil {
		log.Fatalf("failed to create GitHub App client factory: %v", err)
	}

	webhookHandler := handlers.NewWebhookHandler(factory, appLog)

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
