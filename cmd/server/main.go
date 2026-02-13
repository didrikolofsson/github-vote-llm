package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agent"
	"github.com/didrikolofsson/github-vote-llm/internal/cli"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
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

	client := gh.NewClient(cfg.GitHub.Token, log)
	runner := agent.NewRunner(client, &cfg.Agent, log)

	onApproved := func(owner, repo string, issue *gogithub.Issue) {
		repoCfg := cfg.FindRepo(owner, repo)
		if repoCfg == nil {
			log.Infow("no config for repo, skipping", "repo", owner+"/"+repo)
			return
		}
		runner.Run(context.Background(), owner, repo, issue, repoCfg)
	}

	webhook := gh.NewWebhookHandler(cfg, onApproved, log)

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

	// Dev mode: start gh webhook forward with automatic reconnection
	var fwd *webhookForwarder
	if flags.DevMode {
		if flags.DevOwner == "" {
			log.Fatalf("--owner is required in dev mode (e.g. --owner=owner)")
		}
		if flags.DevRepo == "" {
			log.Fatalf("--repo is required in dev mode (e.g. --repo=repo)")
		}
		fwd = newWebhookForwarder(client, flags.DevOwner, flags.DevRepo, cfg.Server.Port, cfg.GitHub.WebhookSecret, log)
		if err := fwd.start(context.Background()); err != nil {
			log.Fatalf("failed to start gh webhook forward: %v", err)
		}
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

	if fwd != nil {
		log.Named("dev").Infow("stopping gh webhook forward")
		fwd.stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		serverLog.Errorw("shutdown error", "error", err)
	}

	serverLog.Infow("stopped")
}

