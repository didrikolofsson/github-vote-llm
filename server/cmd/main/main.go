package main

import (
	"context"
	"log"

	"github.com/didrikolofsson/github-vote-llm/internal/api"
	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	conn, err := pgx.Connect(ctx, env.DATABASE_URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	apiLogger := logger.New().Named("api")
	defer apiLogger.Sync()

	q := store.New(conn)
	userService := services.NewUserService(conn, q)
	userHandlers := handlers.NewUserHandlers(userService, apiLogger)
	authService := services.NewAuthService(conn, q, env.JWT_SECRET)
	authHandlers := handlers.NewAuthHandlers(authService, env.JWT_SECRET)

	router := api.NewRestApiRouter(
		env,
		apiLogger,
		userHandlers,
		authHandlers,
	).Create()

	router.Run(":" + env.PORT)
}
