package main

import (
	"log"
	"net/http"
	"os"

	"github.com/didrikolofsson/github-vote-llm/internal/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func main() {
	if gin.Mode() == gin.DebugMode {
		// Only load .env file in debug mode, expect env vars in production
		if err := godotenv.Load(".env.development"); err != nil {
			log.Fatalf("failed to load .env file: %v", err)
		}
	}

	router := gin.New()
	router.SetTrustedProxies(nil)

	router.GET("/health", healthCheck)

	// Webhook endpoint
	webhookHandlers := handlers.NewWebhookHandler()

	webhooks := router.Group("/github")
	webhooks.Use(middleware.ValidateSignature())
	webhooks.POST("/webhook", webhookHandlers.HandleGithubWebhook)

	router.Run(":" + os.Getenv("PORT"))
}

// log := logger.New()
// defer log.Sync()

// configPath := os.Getenv("CONFIG_PATH")
// if configPath == "" {
// 	configPath = "config.yaml"
// }

// cfg, err := config.Load(configPath)
// if err != nil {
// 	log.Fatalf("failed to load config: %v", err)
// }

// // Auth setup: GitHub App
// clientFactory, err := gh.NewClientFactory(gh.AppConfig{
// 	AppID:          cfg.GitHub.AppID,
// 	PrivateKeyPath: cfg.GitHub.PrivateKeyPath,
// 	WebhookSecret:  cfg.GitHub.WebhookSecret,
// }, log)
// if err != nil {
// 	log.Fatalf("failed to create GitHub App client factory: %v", err)
// }

// // getClient returns a ClientAPI for the given installation.
// getClient := func(installationID int64) gh.ClientAPI {
// 	client, err := gh.NewClient(installationID, clientFactory)
// 	if err != nil {
// 		log.Errorw("failed to create installation client", "installationID", installationID, "error", err)
// 		return nil
// 	}
// 	return client
// }

// onApproved := func(owner, repo string, issue *gogithub.Issue, installationID int64) {
// 	repoCfg := cfg.FindRepo(owner, repo)
// 	if repoCfg == nil {
// 		log.Infow("no config for repo, skipping", "repo", owner+"/"+repo)
// 		return
// 	}
// 	client := getClient(installationID)
// 	if client == nil {
// 		log.Errorw("no GitHub client available, skipping issue", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
// 		return
// 	}
// 	runner := agent.NewRunner(client, &cfg.Agent, log)
// 	runner.Run(context.Background(), owner, repo, issue, repoCfg)
// }

// onVoteCheck := func(owner, repo string, issue *gogithub.Issue, installationID int64) {
// 	repoCfg := cfg.FindRepo(owner, repo)
// 	if repoCfg == nil {
// 		log.Infow("no config for repo, skipping vote check", "repo", owner+"/"+repo)
// 		return
// 	}
// 	client := getClient(installationID)
// 	if client == nil {
// 		log.Errorw("no GitHub client available, skipping vote check", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
// 		return
// 	}
// 	tracker := votes.NewTracker(client, log)
// 	met, err := tracker.CheckAndLabel(context.Background(), owner, repo, issue.GetNumber(), repoCfg)
// 	if err != nil {
// 		log.Errorw("vote check failed", "issue", issue.GetNumber(), "error", err)
// 		return
// 	}
// 	if met {
// 		log.Infow("vote threshold met", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
// 	}
// }

// webhook := gh.NewWebhookHandler(cfg, onApproved, onVoteCheck, log)

// mux := http.NewServeMux()
// mux.Handle("/webhook", webhook)
// mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// 	fmt.Fprintln(w, "ok")
// })

// addr := fmt.Sprintf(":%d", cfg.Server.Port)
// server := &http.Server{
// 	Addr:         addr,
// 	Handler:      mux,
// 	ReadTimeout:  10 * time.Second,
// 	WriteTimeout: 30 * time.Second,
// }

// serverLog := log.Named("server")

// // Graceful shutdown
// ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
// defer stop()

// go func() {
// 	serverLog.Infow("listening", "addr", addr)
// 	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 		log.Fatalf("server error: %v", err)
// 	}
// }()

// <-ctx.Done()
// serverLog.Infow("shutting down")

// shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// defer cancel()
// if err := server.Shutdown(shutdownCtx); err != nil {
// 	serverLog.Errorw("shutdown error", "error", err)
// }

// serverLog.Infow("stopped")
