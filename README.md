# vote-llm

A Go service that automatically implements GitHub feature requests using [Claude Code](https://docs.anthropic.com/en/docs/claude-code) once they reach a community voting threshold.

## How it works

1. A GitHub issue is created with a `feature-request` label
2. Community members vote by adding "+1" reactions to the issue
3. When the vote count reaches the configured threshold, the issue is labeled `approved-for-dev`
4. A GitHub webhook notifies the vote-llm server
5. The server clones the repository, runs Claude Code to implement the feature, and opens a pull request

## Prerequisites

- [Go](https://go.dev/) 1.25+
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated (required for dev mode)
- A GitHub personal access token with repo permissions

## Configuration

Copy and edit the config file:

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
```

Environment variables in values (e.g. `${GITHUB_TOKEN}`) are expanded at load time.

## Usage

Build and run the server:

```bash
go build -o vote-llm ./cmd/server
./vote-llm -config config.yaml
```

### Dev mode

For local development, use dev mode to automatically forward GitHub webhooks to your local server via the GitHub CLI:

```bash
./vote-llm -config config.yaml -dev -owner your-org -repo your-repo
```

### Endpoints

| Path       | Description              |
|------------|--------------------------|
| `/webhook` | GitHub webhook receiver  |
| `/health`  | Health check (returns `ok`) |

## Project structure

```
cmd/server/         Server entry point
internal/
  agent/            Claude Code orchestration
  cli/              CLI flag parsing
  config/           YAML configuration loader
  github/           GitHub API client and webhook handler
  votes/            Vote counting and threshold labeling
```

## License

See [LICENSE](LICENSE) for details.
