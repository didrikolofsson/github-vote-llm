package config

import (
	"fmt"
	"os"

	envpkg "github.com/caarlos0/env/v11"
)

type Environment struct {
	GITHUB_APP_ID      int64  `env:"GITHUB_APP_ID,required"`
	GITHUB_PRIVATE_KEY string `env:"GITHUB_PRIVATE_KEY,required"`
	API_KEY            string `env:"API_KEY,required"`
	WEBHOOK_SECRET     string `env:"WEBHOOK_SECRET,required"`
	DATABASE_URL       string `env:"DATABASE_URL,required"`
	PORT               string `env:"PORT" envDefault:"8080"`
	ANTHROPIC_API_KEY  string `env:"ANTHROPIC_API_KEY,required"`
	WORKSPACE_DIR      string `env:"WORKSPACE_DIR" envDefault:"/tmp/vote-llm-workspaces"`
	JWT_SECRET         string `env:"JWT_SECRET,required"`
}

func LoadEnv() (*Environment, error) {
	// Check if GITHUB_PRIVATE_KEY_PATH is set
	// If set, read the file and set GITHUB_PRIVATE_KEY
	// This is only needed for the local debugger to work
	githubPrivateKeyPath := os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	if githubPrivateKeyPath != "" {
		githubPrivateKey, err := os.ReadFile(githubPrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read github private key file: %w", err)
		}
		os.Setenv("GITHUB_PRIVATE_KEY", string(githubPrivateKey))
	}

	var env Environment
	if err := envpkg.ParseWithOptions(&env, envpkg.Options{
		// https://pkg.go.dev/github.com/caarlos0/env/v11#Options
	}); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}
	return &env, nil
}
