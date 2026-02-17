package main

import (
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	log := logger.New()
	defer log.Sync()

	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	port := ":8080"

	router.Run(port)
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

// // Open SQLite store
// db, err := store.NewStore(cfg.Database.Path, log)
// if err != nil {
// 	log.Fatalf("failed to open store: %v", err)
// }
// defer db.Close()

// // Auth setup: GitHub App or PAT
// var clientFactory *gh.ClientFactory
// var defaultClient gh.ClientAPI

// if cfg.GitHub.AppID != 0 {
// 	// GitHub App mode
// 	factory, err := gh.NewClientFactory(gh.AppConfig{
// 		AppID:          cfg.GitHub.AppID,
// 		PrivateKeyPath: cfg.GitHub.PrivateKeyPath,
// 		WebhookSecret:  cfg.GitHub.WebhookSecret,
// 	}, log)
// 	if err != nil {
// 		log.Fatalf("failed to create GitHub App client factory: %v", err)
// 	}
// 	clientFactory = factory
// 	log.Infow("using GitHub App auth", "appID", cfg.GitHub.AppID)
// } else {
// 	// PAT mode
// 	defaultClient = gh.NewClient(cfg.GitHub.Token, log)
// 	log.Infow("using PAT auth")
// }

// // getClient returns a ClientAPI for the given installation.
// // In PAT mode, always returns the default client.
// // Returns nil if no client could be created.
// getClient := func(installationID int64) gh.ClientAPI {
// 	if clientFactory != nil && installationID != 0 {
// 		client, err := clientFactory.ClientForInstallation(installationID)
// 		if err != nil {
// 			log.Errorw("failed to create installation client", "installationID", installationID, "error", err)
// 			return nil
// 		}
// 		return client
// 	}
// 	if defaultClient == nil {
// 		log.Errorw("no client available (App mode requires installation ID)", "installationID", installationID)
// 	}
// 	return defaultClient
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
// 	runner := agent.NewRunner(client, &cfg.Agent, db, log)
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
