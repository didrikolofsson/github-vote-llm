package config

import (
	"fmt"

	envpkg "github.com/caarlos0/env/v11"
)

type Environment struct {
	GITHUB_CLIENT_ID   string `env:"GITHUB_CLIENT_ID,required"`
	GITHUB_CLIENT_SECRET string `env:"GITHUB_CLIENT_SECRET,required"`
	FRONTEND_URL       string `env:"FRONTEND_URL,required"` // e.g. http://localhost:5173
	TOKEN_ENCRYPTION_KEY string `env:"TOKEN_ENCRYPTION_KEY,required"` // 32-byte hex for AES-256
	API_KEY            string `env:"API_KEY,required"`
	WEBHOOK_SECRET     string `env:"WEBHOOK_SECRET,required"`
	DATABASE_URL       string `env:"DATABASE_URL,required"`
	PORT               string `env:"PORT" envDefault:"8080"`
	ANTHROPIC_API_KEY  string `env:"ANTHROPIC_API_KEY,required"`
	WORKSPACE_DIR      string `env:"WORKSPACE_DIR" envDefault:"/tmp/vote-llm-workspaces"`
	JWT_SECRET         string `env:"JWT_SECRET,required"`
}

func LoadEnv() (*Environment, error) {
	var env Environment
	if err := envpkg.ParseWithOptions(&env, envpkg.Options{
		// https://pkg.go.dev/github.com/caarlos0/env/v11#Options
	}); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}
	return &env, nil
}
