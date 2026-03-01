# github-vote-llm

Community-driven feature development powered by GitHub reactions and Claude Code. Users vote on feature requests with thumbs-up reactions, and when a request is approved, an AI agent automatically implements it and opens a pull request.

## How it works

1. A feature request is filed as a GitHub issue with the `feature-request` label
2. A maintainer labels the issue `approved-for-dev`
3. vote-llm receives the webhook event, adds an `llm-in-progress` label, clones the repo, and runs Claude Code against the issue
4. Claude Code implements the feature; vote-llm commits, pushes, and opens a PR
5. The issue is labeled `llm-pr-created` and a comment links to the PR

## Project structure

```
cmd/main/           HTTP server entry point
db/
  migrations/       SQL migration files (apply with migrate CLI)
  queries/          sqlc query definitions
  sqlc.yaml         sqlc config
internal/
  agent/            Claude Code orchestration (clone, run, commit, PR)
  github/           GitHub App auth and API client
  handlers/         Webhook event handler
  logger/           Structured logging (zap)
  middleware/       Webhook signature validation
  spinner/          Terminal progress spinner
  store/            PostgreSQL store (pgx/v5 + sqlc generated code)
```

## Requirements

- Go 1.25+
- PostgreSQL 14+
- A [GitHub App](https://docs.github.com/en/apps/creating-github-apps) installed on the target repo
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and available on `$PATH`
- [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for running database migrations

## Configuration

All config is via environment variables:

| Variable             | Required | Description                                                                                                      |
| -------------------- | -------- | ---------------------------------------------------------------------------------------------------------------- |
| `GITHUB_APP_ID`      | yes      | GitHub App numeric ID                                                                                            |
| `GITHUB_PRIVATE_KEY` | yes      | PEM bytes as a string                                                                                            |
| `WEBHOOK_SECRET`     | yes      | HMAC secret matching the App's webhook config                                                                    |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI                                                                               |
| `DATABASE_URL`       | yes      | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/dbname`)                                 |
| `PORT`               | no       | HTTP listen port (default: `8080`)                                                                               |
| `WORKSPACE_DIR`      | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`). Point this at a persistent volume in production. |

For local development, put these in `.env.development` — they are loaded automatically when `GIN_MODE=debug`.

## Database

vote-llm requires a PostgreSQL database. Apply the migrations before starting the service for the first time, and again after any upgrade that adds new migrations.

```bash
# Install the migrate CLI (once)
go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply migrations
migrate -source file://db/migrations -database "$DATABASE_URL" up

# Roll back the last migration (if needed)
migrate -source file://db/migrations -database "$DATABASE_URL" down 1
```

### Tables

| Table         | Purpose                                                            |
| ------------- | ------------------------------------------------------------------ |
| `executions`  | Tracks every agent run — status, branch, PR URL, error, timestamps |
| `repo_config` | Per-repo overrides for labels, timeout, budget, and API key        |

`executions` enforces a `UNIQUE(owner, repo, issue_number)` constraint so each issue is processed at most once, even across restarts.

### Per-repo config

Insert a row into `repo_config` to override defaults for a specific repo:

```sql
INSERT INTO repo_config (owner, repo, timeout_minutes, max_budget_usd)
VALUES ('my-org', 'my-repo', 60, 10.00)
ON CONFLICT (owner, repo) DO UPDATE
  SET timeout_minutes = EXCLUDED.timeout_minutes,
      max_budget_usd  = EXCLUDED.max_budget_usd;
```

Any column left `NULL` falls back to the service default (`30` min / `$5.00`).

## Running

```bash
go build -o vote-llm ./cmd/main
export GITHUB_APP_ID=123456
export GITHUB_PRIVATE_KEY=pem-string
export WEBHOOK_SECRET=your-secret
export DATABASE_URL=postgres://user:pass@localhost:5432/vote_llm
./vote-llm
```

## Endpoints

| Path                   | Description                              |
| ---------------------- | ---------------------------------------- |
| `POST /github/webhook` | Receives GitHub webhook events           |
| `GET /health`          | Health check (returns `{"status":"ok"}`) |

## Labels

| Label              | Meaning                                                 |
| ------------------ | ------------------------------------------------------- |
| `feature-request`  | Marks an issue as eligible for automated implementation |
| `approved-for-dev` | Triggers the agent (also requires `feature-request`)    |
| `llm-in-progress`  | Agent is currently working on this issue                |
| `llm-pr-created`   | PR has been opened successfully                         |
| `llm-failed`       | Agent run failed; error details in issue comment        |

## License

See [LICENSE](LICENSE) for details.
