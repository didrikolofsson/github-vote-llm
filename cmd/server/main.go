package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agent"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	gogithub "github.com/google/go-github/v68/github"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	dev := flag.Bool("dev", false, "enable dev mode (auto-starts gh webhook forward)")
	devRepo := flag.String("repo", "", "owner/repo for dev mode webhook forwarding")
	flag.Parse()

	cfg, err := config.Load(*configPath)
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
	if *dev {
		if *devRepo == "" {
			log.Fatal("--repo is required in dev mode (e.g. --repo=owner/repo)")
		}
		ghCmd, err = startWebhookForward(*devRepo, cfg.Server.Port, cfg.GitHub.WebhookSecret)
		if err != nil {
			log.Fatalf("failed to start gh webhook forward: %v", err)
		}
		log.Printf("dev: started gh webhook forward for %s", *devRepo)
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

func startWebhookForward(repo string, port int, secret string) (*exec.Cmd, error) {
	args := []string{
		"webhook", "forward",
		"--repo=" + repo,
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
