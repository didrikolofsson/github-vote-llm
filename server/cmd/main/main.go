package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/didrikolofsson/github-vote-llm/internal/api"
	apihandlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	api_services "github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
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

	st := store.NewPostgresStore(pool)

	runsService := api_services.NewRunsService(st)
	reposService := api_services.NewReposService(st)
	runsHandler := apihandlers.NewRunsHandler(runsService)
	reposHandler := apihandlers.NewReposHandler(reposService)
	boardHandler := apihandlers.NewBoardHandler(st)

	apiHandlers := apihandlers.New(boardHandler, runsHandler, reposHandler)

	api.SetupPublicBoardRouter(router, apiHandlers)
	api.SetupAPIRouter(router, appLog, apiHandlers, env)

	// Serve embedded frontend
	distFS, err := fs.Sub(web.FS, "dist")
	if err != nil {
		log.Fatalf("failed to open embedded dist: %v", err)
	}
	assetsFS, _ := fs.Sub(distFS, "assets")
	router.StaticFS("/assets", http.FS(assetsFS))
	router.GET("/", func(c *gin.Context) {
		data, _ := fs.ReadFile(distFS, "index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	router.GET("/board.html", func(c *gin.Context) {
		data, _ := fs.ReadFile(distFS, "board.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/assets/") {
			c.Status(http.StatusNotFound)
			return
		}
		if path == "/board" || strings.HasPrefix(path, "/board/") {
			data, _ := fs.ReadFile(distFS, "board.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}
		data, _ := fs.ReadFile(distFS, "index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
