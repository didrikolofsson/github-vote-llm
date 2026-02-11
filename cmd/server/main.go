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
		ghCmd, err = startWebhookForward(client, flags.DevOwner, flags.DevRepo, cfg.Server.Port, cfg.GitHub.WebhookSecret)
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
