package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub   GitHubConfig   `yaml:"github"`
	Server   ServerConfig   `yaml:"server"`
	Repos    []RepoConfig   `yaml:"repos"`
	Agent    AgentConfig    `yaml:"agent"`
	Database DatabaseConfig `yaml:"database"`
}

type GitHubConfig struct {
	Token          string `yaml:"token"`
	WebhookSecret  string `yaml:"webhook_secret"`
	AppID          int64  `yaml:"app_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type RepoConfig struct {
	Owner          string      `yaml:"owner"`
	Name           string      `yaml:"name"`
	Labels         LabelConfig `yaml:"labels"`
	VoteThreshold  int         `yaml:"vote_threshold"`
}

type LabelConfig struct {
	FeatureRequest string `yaml:"feature_request"`
	Approved       string `yaml:"approved"`
	InProgress     string `yaml:"in_progress"`
	Done           string `yaml:"done"`
	Candidate      string `yaml:"candidate"`
	Failed         string `yaml:"failed"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AgentConfig struct {
	Command        string   `yaml:"command"`
	MaxTurns       int      `yaml:"max_turns"`
	MaxBudgetUSD   float64  `yaml:"max_budget_usd"`
	AllowedTools   []string `yaml:"allowed_tools"`
	WorkspaceDir   string   `yaml:"workspace_dir"`
	TimeoutMinutes int      `yaml:"timeout_minutes"`
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// Load reads the config file and expands environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	expanded := envVarPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		varName := envVarPattern.FindSubmatch(match)[1]
		return []byte(os.Getenv(string(varName)))
	})

	var cfg Config
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	hasToken := c.GitHub.Token != ""
	hasApp := c.GitHub.AppID != 0 && c.GitHub.PrivateKeyPath != ""
	if !hasToken && !hasApp {
		return fmt.Errorf("github: either token (set GITHUB_TOKEN) or app_id + private_key_path is required")
	}
	if c.GitHub.WebhookSecret == "" {
		return fmt.Errorf("github.webhook_secret is required (set WEBHOOK_SECRET)")
	}
	if c.Database.Path == "" {
		c.Database.Path = "vote-llm.db"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if len(c.Repos) == 0 {
		return fmt.Errorf("at least one repo must be configured")
	}
	for i, r := range c.Repos {
		if r.Owner == "" || r.Name == "" {
			return fmt.Errorf("repos[%d]: owner and name are required", i)
		}
		if r.Labels.Approved == "" {
			c.Repos[i].Labels.Approved = "approved-for-dev"
		}
		if r.Labels.FeatureRequest == "" {
			c.Repos[i].Labels.FeatureRequest = "feature-request"
		}
		if r.Labels.InProgress == "" {
			c.Repos[i].Labels.InProgress = "llm-in-progress"
		}
		if r.Labels.Done == "" {
			c.Repos[i].Labels.Done = "llm-pr-created"
		}
		if r.Labels.Candidate == "" {
			c.Repos[i].Labels.Candidate = "candidate"
		}
		if r.Labels.Failed == "" {
			c.Repos[i].Labels.Failed = "llm-failed"
		}
		if r.VoteThreshold == 0 {
			c.Repos[i].VoteThreshold = 5
		}
	}
	if c.Agent.Command == "" {
		c.Agent.Command = "claude"
	}
	if c.Agent.MaxTurns == 0 {
		c.Agent.MaxTurns = 25
	}
	if c.Agent.MaxBudgetUSD == 0 {
		c.Agent.MaxBudgetUSD = 5.00
	}
	if c.Agent.WorkspaceDir == "" {
		c.Agent.WorkspaceDir = "/tmp/vote-llm-workspaces"
	}
	if c.Agent.TimeoutMinutes == 0 {
		c.Agent.TimeoutMinutes = 30
	}
	return nil
}

// FindRepo returns the config for a given owner/repo, or nil if not configured.
func (c *Config) FindRepo(owner, name string) *RepoConfig {
	for i := range c.Repos {
		if c.Repos[i].Owner == owner && c.Repos[i].Name == name {
			return &c.Repos[i]
		}
	}
	return nil
}
