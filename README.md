# github-vote-llm

Community-driven feature development powered by GitHub reactions and Claude Code. Users vote on feature requests with thumbs-up reactions, and when a request is approved, an AI agent automatically implements it and opens a pull request.

## How it works

1. A feature request is filed as a GitHub issue
2. Community members vote by adding +1 reactions
3. A maintainer labels the issue as approved (e.g. `approved-for-dev`)
4. vote-llm receives the webhook event, adds an `llm-in-progress` label, clones the repo, and runs Claude Code against the issue
5. Claude Code implements the feature, vote-llm commits, pushes, and opens a PR
6. The issue is labeled `llm-pr-created` and a comment links to the PR

## Project structure

```
cmd/server/       HTTP server entry point
internal/
  agent/          Claude Code orchestration (clone, run, commit, PR)
  cli/            CLI flag parsing
  config/         YAML configuration loading with env-var expansion
  github/         GitHub API client and webhook handler
  logger/         Structured logging with colored output (zap)
  votes/          Reaction-based vote counting (not currently wired into main flow)
config.yaml       Example configuration
```

## Requirements

- Go 1.25+
- A GitHub personal access token with repo permissions
- A webhook secret for signature verification
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and available on `$PATH`
- [GitHub CLI](https://cli.github.com/) (`gh`) for dev mode webhook forwarding

## Configuration

Copy and edit `config.yaml`:

```yaml
github:
  token: "${GITHUB_TOKEN}"
  webhook_secret: "${WEBHOOK_SECRET}"

server:
  port: 8080

repos:
  - owner: "your-org"
    name: "your-repo"
    labels:
      feature_request: "feature-request"
      approved: "approved-for-dev"
      in_progress: "llm-in-progress"
      done: "llm-pr-created"
    vote_threshold: 5

agent:
  command: "claude"
  max_turns: 25
  max_budget_usd: 5.00
  allowed_tools:
    - "Read"
    - "Edit"
    - "Write"
    - "Bash"
    - "Glob"
    - "Grep"
  workspace_dir: "/tmp/vote-llm-workspaces"
  timeout_minutes: 30
```

Environment variables referenced with `${VAR}` syntax are expanded at load time.

## Usage

Build and run:

```bash
go build -o vote-llm ./cmd/server
export GITHUB_TOKEN="ghp_..."
export WEBHOOK_SECRET="your-secret"
./vote-llm --config=config.yaml
```

### Dev mode

Dev mode uses `gh webhook forward` to tunnel GitHub webhooks to your local machine:

```bash
./vote-llm --dev --owner=your-org --repo=your-repo
```

## Endpoints

| Path       | Description                    |
|------------|--------------------------------|
| `/webhook` | Receives GitHub webhook events |
| `/health`  | Health check (returns `ok`)    |

## License

See [LICENSE](LICENSE) for details.
