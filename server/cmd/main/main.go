package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	apimw "github.com/didrikolofsson/github-vote-llm/internal/api"
	apihandlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/didrikolofsson/github-vote-llm/internal/webhook"
	"github.com/didrikolofsson/github-vote-llm/web"
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
	boardHandler := apihandlers.NewBoardHandler(st)

	// Public board routes (no API key required)
	board := router.Group("/board")
	board.GET("/:owner/:repo/proposals", boardHandler.ListProposals)
	board.POST("/:owner/:repo/proposals", boardHandler.CreateProposal)
	board.POST("/:owner/:repo/proposals/:id/vote", boardHandler.VoteProposal)
	board.GET("/:owner/:repo/proposals/:id/comments", boardHandler.ListComments)
	board.POST("/:owner/:repo/proposals/:id/comments", boardHandler.CreateComment)

	api.Use(apimw.ValidateAPIKey(env.API_KEY))

	api.GET("/runs", runsHandler.List)
	api.POST("/runs", runsHandler.Create)
	api.GET("/runs/:id", runsHandler.Get)
	api.POST("/runs/:id/retry", runsHandler.Retry)
	api.POST("/runs/:id/cancel", runsHandler.Cancel)
	api.GET("/repos", reposHandler.List)
	api.GET("/repos/:owner/:repo/config", reposHandler.GetConfig)
	api.PUT("/repos/:owner/:repo/config", reposHandler.UpdateConfig)
	api.DELETE("/repos/:owner/:repo/config", reposHandler.DeleteConfig)
	api.GET("/repos/:owner/:repo/roadmap", reposHandler.ListRoadmapItems)
	api.PATCH("/repos/:owner/:repo/proposals/:id", reposHandler.UpdateProposalStatus)

	// Serve board SPA at /board/*
	boardFS, _ := fs.Sub(web.FS, "dist")
	router.GET("/board/:owner/:repo", func(c *gin.Context) {
		data, err := fs.ReadFile(boardFS, "board.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// Serve SPA at /ui
	uiFS, _ := fs.Sub(web.FS, "dist")
	fileServer := http.StripPrefix("/ui", http.FileServer(http.FS(uiFS)))
	router.GET("/ui", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})
	router.GET("/ui/*filepath", func(c *gin.Context) {
		filePath := strings.TrimPrefix(c.Param("filepath"), "/")
		if filePath == "" {
			filePath = "index.html"
		}
		f, err := uiFS.Open(filePath)
		if err != nil {
			// SPA fallback: serve index.html for client-side routing
			data, readErr := fs.ReadFile(uiFS, "index.html")
			if readErr != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}
		f.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
