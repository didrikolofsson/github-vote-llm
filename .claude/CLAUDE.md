# github-vote-llm

A Go service that automates feature implementation for GitHub repos. When an issue receives an approval label, it runs **Claude Code** (`claude` CLI) to implement the feature, commit changes, and open a PR.

## How It Works

1. **Webhook** — Listens for GitHub `issues` webhook events at `/v1/api/github/webhook`.
2. **Approval trigger** — When an issue gets the `approved-for-dev` label (requires `feature-request` label), the agent is triggered.
3. **Agent flow** — For each approved issue:
   - Guards: skips if already has `llm-in-progress` label or lacks `feature-request` label
   - Adds `llm-in-progress` label, removes `approved-for-dev` label
   - Clones/updates the repo into a workspace (using installation token for auth)
   - Creates a branch `vote-llm/issue-{n}-{slug}` from the repo's default branch
   - Runs `claude -p` with the issue as the prompt (30 min timeout, $5 budget cap)
   - Commits, pushes (with `--force-with-lease`), and creates a PR
   - Adds `llm-pr-created` label, removes in-progress, comments with PR link
   - On failure: comments with error details, adds `llm-failed` label, removes in-progress label

## Project Structure

```
cmd/main/main.go                        # Entry point: env var config, Gin router, GitHub App auth
db/
  migrations/                           # SQL migration files (apply with migrate CLI)
  queries/                              # sqlc query definitions
  sqlc.yaml                             # sqlc config
internal/
  agent/runner.go                       # Orchestrates Claude Code: clone, branch, run claude, commit, PR
  agent/runner_test.go                  # Agent tests
  api/
    middleware.go                       # API key validation middleware (X-Api-Key header)
    handlers/runs.go                    # REST handlers: GET/POST /runs, GET /runs/:id, retry, cancel
    handlers/runs_test.go               # Runs handler unit tests
    handlers/repos.go                   # REST handlers: GET /repos, GET/PUT /repos/:owner/:repo/config
    handlers/repos_test.go              # Repos handler unit tests
    services/runs.go                    # RunsService: business logic over store for executions
    services/repos.go                   # ReposService: business logic over store for repo configs
  config/environment.go                 # Env var parsing via caarlos0/env
  github/auth.go                        # GitHub App auth: ClientFactory (installation tokens via go-githubauth)
  github/client.go                      # ClientAPI interface + App-based Client implementation
  helpers/helpers.go                    # Shared utilities (e.g. float64↔pgtype.Numeric conversion)
  logger/logger.go                      # Structured logging with colored output (zap)
  logger/logger_test.go                 # Logger tests
  spinner/spinner.go                    # Terminal progress spinner
  store/store.go                        # Store interface + PostgresStore (pgx/v5 + sqlc)
  store/models.go                       # Model types (Execution, RepoConfig, IssueVote)
  store/mock_store.go                   # MockStore for tests
  store/executions.sql.go               # sqlc-generated: execution queries
  store/repo_config.sql.go              # sqlc-generated: repo_config queries
  store/issue_votes.sql.go              # sqlc-generated: issue_votes queries
  webhook/webhook.go                    # Webhook handler: issues/labeled event, approval callback
  webhook/webhook_test.go               # Webhook handler tests (MockStore, deduplication paths)
  webhook/middleware.go                 # HMAC-SHA256 webhook signature validation
openapi.yaml                            # Hand-maintained OpenAPI 3.1 spec (source of truth for SDK generation)
```

## Configuration

All configuration via environment variables (no config file):

| Variable             | Required | Description                                  |
| -------------------- | -------- | -------------------------------------------- |
| `GITHUB_APP_ID`      | yes      | GitHub App numeric ID                        |
| `GITHUB_PRIVATE_KEY` | yes      | PEM bytes as a string (no file path option)  |
| `WEBHOOK_SECRET`     | yes      | HMAC secret for webhook signature validation |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI           |
| `API_KEY`            | yes      | API key for REST endpoints (X-Api-Key header) |
| `DATABASE_URL`       | yes      | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/dbname`) |
| `PORT`               | no       | HTTP listen port (default: `8080`)           |
| `WORKSPACE_DIR`      | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`). Point this at a persistent volume in production. |

In `GIN_MODE=debug`, env vars are loaded from `.env.development` via godotenv.

## Auth

**GitHub App only.** Set `GITHUB_APP_ID` + `GITHUB_PRIVATE_KEY`. Creates per-installation clients with short-lived tokens. Git clone/push uses `x-access-token:{token}@github.com` URLs.

**REST API.** All routes under `/v1/api` (except the webhook and health check) require `X-Api-Key: {API_KEY}` header. Validated by `internal/api/middleware.go`.

## Dependencies

- **claude** CLI — The agent command; must be installed and configured.
- **git** — Used by agent for clone, branch, commit, push.
- **github.com/jferrl/go-githubauth** — GitHub App JWT + installation token auth.
- **github.com/gin-gonic/gin** — HTTP router.
- **go.uber.org/zap** — Structured logging.
- **github.com/jackc/pgx/v5** — PostgreSQL driver and connection pool.
- **github.com/caarlos0/env/v11** — Env var parsing into structs.
- **sqlc** — Generates type-safe Go from SQL queries (`db/queries/`). Run `sqlc generate` to regenerate `internal/store/*.sql.go`.
- **golang-migrate** CLI — Applies migrations in `db/migrations/`.

## Key Implementation Details

- **ClientAPI interface**: All GitHub operations go through `github.ClientAPI`. One implementation: `Client` (GitHub App installation tokens).
- **Webhook validation**: HMAC-SHA256 signature checked in `webhook.ValidateSignature()` before the handler sees the payload. Parsing via `gh.ParseWebHook` from go-github.
- **Duplicate prevention**: DB-enforced via `UNIQUE(owner, repo, issue_number)` on the `executions` table — `CreateExecution` returns an error on conflict, preventing re-processing even across restarts.
- **Execution lifecycle**: `executions` rows transition through `pending → in_progress → success/failed/cancelled`; runner calls `SetInProgress`, `SetSuccess`, or `SetFailed` at each step.
- **Per-repo config**: `GetRepoConfig` looks up `repo_config` for a (owner, repo) pair and returns a fully-resolved `RepoConfigModel` (no nullable fields — defaults applied in the store layer). `timeout_minutes`, `max_budget_usd`, all five labels (`label_approved`, `label_feature_request`, `label_in_progress`, `label_done`, `label_failed`), `vote_threshold`, and `anthropic_api_key` all override service-level defaults. `NULL` DB columns fall back to defaults.
- **Label state machine**: `feature-request` → (manual) → `approved-for-dev` → `llm-in-progress` → `llm-pr-created` or `llm-failed`.
- **Feature-request guard**: `approved-for-dev` label is only processed if the issue also has `feature-request`.
- **Agent workspace**: `$WORKSPACE_DIR/{owner}/{repo}/repo` — reused across runs; `git fetch --all --prune` before new work.
- **Branch naming**: `vote-llm/issue-{number}-{slugified-title}` (slug max 40 chars).
- **Claude invocation**: `claude -p {prompt} --output-format json --allowedTools {tools} --max-turns 25 --max-budget-usd 5.00 --no-session-persistence`
- **Existing PR handling**: If PR creation fails, `FindPullRequestByHead` looks up an existing open PR for the branch.
- **Default branch**: Uses `GetDefaultBranch()` to discover the repo's default branch (not hardcoded to `main`).
- **Logging**: Structured logging via `go.uber.org/zap` with colored console output and component hierarchy (e.g. `webhook`, `agent`).
- **OpenAPI spec**: `openapi.yaml` at the repo root is hand-maintained (OpenAPI 3.1). It is the source of truth for SDK generation via heyAPI. Update it when adding or changing API endpoints.
- **REST API layer**: Handlers in `internal/api/handlers/` use plain Gin (`*gin.Context`). Services in `internal/api/services/` sit between handlers and the store. Handlers are unit-tested with MockStore via `httptest`.
