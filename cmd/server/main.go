package main

import (
	"context"
	"fmt"
	"log"
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
	gogithub "github.com/google/go-github/v68/github"
)

func main() {
	flags, err := cli.ParseFlags()
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	client := gh.NewClient(cfg.GitHub.Token)
	runner := agent.NewRunner(client, &cfg.Agent)

	onApproved := func(owner, repo string, issue *gogithub.Issue) {
		repoCfg := cfg.FindRepo(owner, repo)
		if repoCfg == nil {
			log.Printf("no config for %s/%s, skipping", owner, repo)
			return
		}
		runner.Run(context.Background(), owner, repo, issue, repoCfg)
	}

	webhook := gh.NewWebhookHandler(cfg, onApproved)

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

	// Dev mode: start gh webhook forward as a subprocess
	var ghCmd *exec.Cmd
	if flags.DevMode {
		if flags.DevOwner == "" {
			log.Fatal("--owner is required in dev mode (e.g. --owner=owner)")
		}
		if flags.DevRepo == "" {
			log.Fatal("--repo is required in dev mode (e.g. --repo=repo)")
		}
		ghCmd, err = startWebhookForward(client, flags.DevOwner, flags.DevRepo, cfg.Server.Port, cfg.GitHub.WebhookSecret)
		if err != nil {
			log.Fatalf("failed to start gh webhook forward: %v", err)
		}
		log.Printf("dev: started gh webhook forward for %s", flags.DevRepo)
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server: listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-stop
	log.Println("server: shutting down...")

	if ghCmd != nil && ghCmd.Process != nil {
		log.Println("dev: stopping gh webhook forward...")
		ghCmd.Process.Signal(syscall.SIGTERM)
		ghCmd.Wait()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server: shutdown error: %v", err)
	}

	log.Println("server: stopped")
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
