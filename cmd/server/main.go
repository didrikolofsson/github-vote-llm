package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agent"
	"github.com/didrikolofsson/github-vote-llm/internal/cli"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/didrikolofsson/github-vote-llm/internal/votes"
	gogithub "github.com/google/go-github/v68/github"
)

func main() {
	log := logger.New()
	defer log.Sync()

	flags, err := cli.ParseFlags()
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Open SQLite store
	db, err := store.NewStore(cfg.Database.Path, log)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	defer db.Close()

	// Auth setup: GitHub App or PAT
	var clientFactory *gh.ClientFactory
	var defaultClient gh.ClientAPI

	if cfg.GitHub.AppID != 0 {
		// GitHub App mode
		factory, err := gh.NewClientFactory(gh.AppConfig{
			AppID:          cfg.GitHub.AppID,
			PrivateKeyPath: cfg.GitHub.PrivateKeyPath,
			WebhookSecret:  cfg.GitHub.WebhookSecret,
		}, log)
		if err != nil {
			log.Fatalf("failed to create GitHub App client factory: %v", err)
		}
		clientFactory = factory
		log.Infow("using GitHub App auth", "appID", cfg.GitHub.AppID)
	} else {
		// PAT mode
		defaultClient = gh.NewClient(cfg.GitHub.Token, log)
		log.Infow("using PAT auth")
	}

	// getClient returns a ClientAPI for the given installation.
	// In PAT mode, always returns the default client.
	// Returns nil if no client could be created.
	getClient := func(installationID int64) gh.ClientAPI {
		if clientFactory != nil && installationID != 0 {
			client, err := clientFactory.ClientForInstallation(installationID)
			if err != nil {
				log.Errorw("failed to create installation client", "installationID", installationID, "error", err)
				return nil
			}
			return client
		}
		if defaultClient == nil {
			log.Errorw("no client available (App mode requires installation ID)", "installationID", installationID)
		}
		return defaultClient
	}

	onApproved := func(owner, repo string, issue *gogithub.Issue, installationID int64) {
		repoCfg := cfg.FindRepo(owner, repo)
		if repoCfg == nil {
			log.Infow("no config for repo, skipping", "repo", owner+"/"+repo)
			return
		}
		client := getClient(installationID)
		if client == nil {
			log.Errorw("no GitHub client available, skipping issue", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
			return
		}
		runner := agent.NewRunner(client, &cfg.Agent, db, log)
		runner.Run(context.Background(), owner, repo, issue, repoCfg)
	}

	onVoteCheck := func(owner, repo string, issue *gogithub.Issue, installationID int64) {
		repoCfg := cfg.FindRepo(owner, repo)
		if repoCfg == nil {
			log.Infow("no config for repo, skipping vote check", "repo", owner+"/"+repo)
			return
		}
		client := getClient(installationID)
		if client == nil {
			log.Errorw("no GitHub client available, skipping vote check", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
			return
		}
		tracker := votes.NewTracker(client, log)
		met, err := tracker.CheckAndLabel(context.Background(), owner, repo, issue.GetNumber(), repoCfg)
		if err != nil {
			log.Errorw("vote check failed", "issue", issue.GetNumber(), "error", err)
			return
		}
		if met {
			log.Infow("vote threshold met", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
		}
	}

	webhook := gh.NewWebhookHandler(cfg, onApproved, onVoteCheck, log)

	mux := http.NewServeMux()
	mux.Handle("/webhook", webhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	serverLog := log.Named("server")

	// Dev mode: start gh webhook forward as a subprocess
	var ghCmd *exec.Cmd
	if flags.DevMode {
		devLog := log.Named("dev")
		if flags.DevOwner == "" {
			log.Fatalf("--owner is required in dev mode (e.g. --owner=owner)")
		}
		if flags.DevRepo == "" {
			log.Fatalf("--repo is required in dev mode (e.g. --repo=repo)")
		}
		// Dev mode uses PAT client for webhook forward cleanup
		patClient := gh.NewClient(cfg.GitHub.Token, log)
		ghCmd, err = startWebhookForward(patClient, flags.DevOwner, flags.DevRepo, cfg.Server.Port, cfg.GitHub.WebhookSecret)
		if err != nil {
			log.Fatalf("failed to start gh webhook forward: %v", err)
		}
		devLog.Infow("started gh webhook forward", "repo", flags.DevRepo)
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		serverLog.Infow("listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-stop
	serverLog.Infow("shutting down")

	if ghCmd != nil && ghCmd.Process != nil {
		log.Named("dev").Infow("stopping gh webhook forward")
		ghCmd.Process.Signal(syscall.SIGTERM)
		ghCmd.Wait()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		serverLog.Errorw("shutdown error", "error", err)
	}

	serverLog.Infow("stopped")
}

func startWebhookForward(client *gh.Client, owner, repo string, port int, secret string) (*exec.Cmd, error) {
	if err := client.RemoveLocalRepoWebhooks(context.Background(), owner, repo, port); err != nil {
		return nil, fmt.Errorf("remove local repo webhooks: %w", err)
	}

	args := []string{
		"webhook", "forward",
		"--repo=" + fmt.Sprintf("%s/%s", owner, repo),
		"--events=issues,issue_comment",
		fmt.Sprintf("--url=http://localhost:%d/webhook", port),
		"--secret=" + secret,
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start gh: %w", err)
	}

	return cmd, nil
}
