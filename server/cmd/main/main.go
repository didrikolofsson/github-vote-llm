package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agents/claude"
	"github.com/didrikolofsson/github-vote-llm/internal/api"
	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/workers"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
)

func main() {
	appLogger := logger.New().Named("main")
	defer appLogger.Sync()

	if gin.Mode() == gin.DebugMode {
		if err := godotenv.Load(); err != nil {
			appLogger.Fatalf("failed to load .env file: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env, err := config.LoadEnv()
	if err != nil {
		appLogger.Fatalf("failed to load environment: %v", err)
	}

	db, err := pgxpool.New(ctx, env.DATABASE_URL)
	if err != nil {
		appLogger.Fatalf("failed to create database pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		appLogger.Fatalf("failed to reach database: %v", err)
	}

	w := river.NewWorkers()
	jc, err := jobs.NewClient(jobs.JobClientDeps{DB: db, Workers: w})
	if err != nil {
		appLogger.Fatalf("failed to create job client: %v", err)
	}
	claudeRunner := claude.NewClaudeRunner(claude.NewClaudeRunnerParams{
		ApiKey: env.ANTHROPIC_API_KEY,
		Logger: appLogger,
	})

	q := store.New(db)
	eventHub := hub.NewHub()
	s := services.New(services.ServicesDeps{
		DB: db, Queries: q, Env: env, JobClient: jc, Hub: eventHub,
		AgentRunner: claudeRunner,
	})

	workers.Register(w, workers.RegisterWorkersDeps{Services: s, Env: env, Logger: appLogger})
	if err := jc.Start(ctx); err != nil {
		appLogger.Fatalf("failed to start job client: %v", err)
	}

	apiLogger := logger.New().Named("api")
	defer apiLogger.Sync()

	h := handlers.New(handlers.NewHandlersDeps{Services: s, Logger: apiLogger, Hub: eventHub})
	router := api.New(h, apiLogger, env)

	srv := &http.Server{
		Addr:    ":" + env.PORT,
		Handler: router,
	}

	go func() {
		appLogger.Infof("listening on :%s", env.PORT)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	appLogger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Errorf("HTTP shutdown error: %v", err)
	}

	if err := jc.Stop(shutdownCtx); err != nil {
		appLogger.Errorf("job client shutdown error: %v", err)
	}

	appLogger.Info("shutdown complete")
}
