# github-vote-llm

A Go service that automates feature implementation for GitHub repos. When an issue receives an approval label, it runs **Claude Code** (`claude` CLI) to implement the feature, commit changes, and open a PR.

## How It Works

1. **Webhook** — Listens for GitHub `issues` and `issue_comment` webhook events.
2. **Vote counting** — When a `feature-request` issue gets comments or reactions, votes (+1 reactions) are checked. If the threshold is met, a `candidate` label is added.
3. **Approval trigger** — When an issue gets the configured "approved" label (e.g. `approved-for-dev`), the agent is triggered (requires `feature-request` label).
4. **Agent flow** — For each approved issue:
   - Checks SQLite store for idempotency (skips if already processing/completed)
   - Creates execution record, adds `llm-in-progress` label, removes `approved` label
   - Clones/updates the repo into a workspace (using installation token for auth)
   - Creates a branch `vote-llm/issue-{n}-{slug}` from the repo's default branch
   - Runs `claude -p` with the issue as the prompt (with configurable timeout, default 30 min)
   - Commits, pushes (with `--force-with-lease`), and creates a PR
   - Adds `llm-pr-created` label, removes in-progress, comments with PR link
   - On failure: comments with error details, adds `llm-failed` label, removes in-progress label

## Project Structure

```
cmd/server/main.go          # Entry point: flags, config, HTTP server, auth mode selection, webhook forward (dev mode)
internal/
  agent/runner.go           # Orchestrates Claude Code: clone, branch, run claude, commit, PR (with store + ClientAPI)
  cli/flag.go               # CLI flags (--config, --dev, --owner, --repo)
  config/config.go          # YAML config loading, env var expansion, validation
  github/auth.go            # GitHub App auth: ClientFactory, AppClient (installation tokens via go-githubauth)
  github/client.go          # ClientAPI interface + PAT-based Client implementation
  github/webhook.go         # Webhook handler: issues/labeled + issue_comment events, vote check + approval callbacks
  logger/logger.go          # Structured logging with colored output (zap)
  logger/logger_test.go     # Logger tests
  store/store.go            # SQLite persistence: execution records for idempotency
  store/migrations.go       # Database schema migrations
  store/store_test.go       # Store tests (7 tests)
  votes/tracker.go          # Vote counting (+1 reactions), wired into webhook handler
```

## Configuration

`config.yaml`:

- **github.token** — PAT auth (env var: `${GITHUB_TOKEN}`)
- **github.webhook_secret** — Webhook secret (env var: `${WEBHOOK_SECRET}`)
- **github.app_id**, **github.private_key_path** — GitHub App auth (alternative to token)
- **database.path** — SQLite database path (default: `vote-llm.db`)
- **repos** — Per-repo: owner, name, labels (feature_request, approved, in_progress, done, candidate, failed), vote_threshold
- **agent** — `command` (e.g. `claude`), max_turns, max_budget_usd, allowed_tools, workspace_dir, timeout_minutes

## Auth Modes

- **PAT mode** (default for dev): Set `github.token` in config. All API calls use the same token.
- **GitHub App mode**: Set `github.app_id` + `github.private_key_path`. Creates per-installation clients with short-lived tokens. Git clone/push uses `x-access-token:{token}@github.com` URLs.

## Dev Mode

For local development without a real webhook:

```bash
go run ./cmd/server --dev --owner=didrikolofsson --repo=github-vote-llm
```

- Starts `gh webhook forward` as a subprocess to forward GitHub webhooks to `localhost`
- **Before** creating a new forward, removes existing `gh webhook forward` webhooks (avoids "Hook already exists" when a previous run was killed)
- On shutdown (SIGINT/SIGTERM), sends SIGTERM to the `gh` process and shuts down the server
- Dev mode always uses PAT auth for webhook forward cleanup

## Dependencies

- **gh** CLI — Required for dev mode (`gh webhook forward`). Must be logged in.
- **claude** CLI — The agent command; must be installed and configured.
- **git** — Used by agent for clone, branch, commit, push.
- **github.com/jferrl/go-githubauth** — GitHub App JWT + installation token auth.
- **modernc.org/sqlite** — Pure-Go SQLite driver (no CGO).

## Key Implementation Details

- **ClientAPI interface**: All GitHub operations go through `github.ClientAPI`. Two implementations: `Client` (PAT) and `AppClient` (GitHub App installation tokens).
- **Webhook validation**: Uses `gh.ValidatePayload` and `gh.ParseWebHook` from go-github.
- **Duplicate prevention**: Webhook handler skips issues that already have the `in-progress` label. SQLite store provides idempotency across restarts.
- **Label state machine**: `feature-request` → (votes) → `candidate` → (manual) → `approved-for-dev` → `llm-in-progress` → `llm-pr-created` or `llm-failed`.
- **Feature-request guard**: `approved-for-dev` label is only processed if the issue also has `feature-request`.
- **Vote wiring**: `IssueCommentEvent` on feature-request issues and `feature-request` label addition trigger vote checks.
- **Agent workspace**: `{workspace_dir}/{owner}/{repo}/repo` — reused across runs; `git fetch` before new work.
- **Branch naming**: `vote-llm/issue-{number}-{slugified-title}` (slug max 40 chars).
- **Claude invocation**: `claude -p {prompt} --output-format json --allowedTools {tools} --max-turns {n} --max-budget-usd {n} --no-session-persistence`
- **Existing PR handling**: If PR creation fails, `FindPullRequestByHead` looks up an existing open PR for the branch.
- **Default branch**: Uses `GetDefaultBranch()` to discover the repo's default branch (not hardcoded to `main`).
- **Logging**: Structured logging via `go.uber.org/zap` with colored console output and component hierarchy (e.g. `server.http`).
- **github/client.RemoveLocalRepoWebhooks**: Deletes hooks with URL `https://webhook-forwarder.github.com/hook` (gh webhook forward); used to clean up before creating a new forward.
