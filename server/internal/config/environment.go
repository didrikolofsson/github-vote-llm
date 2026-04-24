package config

import (
	"fmt"

	envpkg "github.com/caarlos0/env/v11"
)

type Environment struct {
	GITHUB_APP_ID             int64  `env:"GITHUB_APP_ID,required"`
	GITHUB_APP_SLUG           string `env:"GITHUB_APP_SLUG,required"`
	GITHUB_APP_PRIVATE_KEY    string `env:"GITHUB_APP_PRIVATE_KEY,required"` // PEM or base64-encoded PEM
	GITHUB_APP_WEBHOOK_SECRET string `env:"GITHUB_APP_WEBHOOK_SECRET,required"`
	FRONTEND_URL              string `env:"FRONTEND_URL,required"`        // e.g. http://localhost:5173
	SERVER_URL                string `env:"SERVER_URL,required"`          // e.g. http://localhost:8080 — base URL the browser uses to reach the API
	API_KEY                   string `env:"API_KEY,required"`
	DATABASE_URL              string `env:"DATABASE_URL,required"`
	PORT                      string `env:"PORT" envDefault:"8080"`
	ANTHROPIC_API_KEY         string `env:"ANTHROPIC_API_KEY,required"`
	WORKSPACE_DIR             string `env:"WORKSPACE_DIR" envDefault:"/tmp/vote-llm-workspaces"`
	JWT_SECRET                string `env:"JWT_SECRET,required"`
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
