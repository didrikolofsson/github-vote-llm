package main

import (
	"context"
	"log"

	"github.com/didrikolofsson/github-vote-llm/internal/api"
	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/river"
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
	pool, err := pgxpool.New(ctx, env.DATABASE_URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	apiLogger := logger.New().Named("api")
	defer apiLogger.Sync()

	q := store.New(pool)
	handlers := handlers.NewHandlerCollection(pool, q, env, apiLogger)

	rc := river.NewRiverClient(ctx, pool)
	rc.Start(ctx)
	defer rc.Stop(ctx)

	router := api.NewRestApiRouter(
		env,
		apiLogger,
		handlers,
		rc,
	).Create()

	router.Run(":" + env.PORT)
}
