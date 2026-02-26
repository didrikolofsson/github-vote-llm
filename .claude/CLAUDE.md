# github-vote-llm

A Go service that automates feature implementation for GitHub repos. When an issue receives an approval label, it runs **Claude Code** (`claude` CLI) to implement the feature, commit changes, and open a PR.

## How It Works

1. **Webhook** — Listens for GitHub `issues` webhook events at `/github/webhook`.
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
cmd/main/main.go              # Entry point: env var config, Gin router, GitHub App auth
internal/
  agent/runner.go             # Orchestrates Claude Code: clone, branch, run claude, commit, PR
  agent/runner_test.go        # Agent tests
  github/auth.go              # GitHub App auth: ClientFactory (installation tokens via go-githubauth)
  github/client.go            # ClientAPI interface + App-based Client implementation
  handlers/webhook.go         # Webhook handler: issues/labeled event, approval callback
  logger/logger.go            # Structured logging with colored output (zap)
  logger/logger_test.go       # Logger tests
  middleware/middleware.go     # HMAC-SHA256 webhook signature validation
  spinner/spinner.go          # Terminal progress spinner
```

## Configuration

All configuration via environment variables (no config file):

| Variable             | Required | Description                                  |
| -------------------- | -------- | -------------------------------------------- |
| `GITHUB_APP_ID`      | yes      | GitHub App numeric ID                        |
| `GITHUB_PRIVATE_KEY` | yes      | PEM bytes as a string (no file path option)  |
| `WEBHOOK_SECRET`     | yes      | HMAC secret for webhook signature validation |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI           |
| `PORT`               | no       | HTTP listen port (default: `8080`)           |

In `GIN_MODE=debug`, env vars are loaded from `.env.development` via godotenv.

## Auth

**GitHub App only.** Set `GITHUB_APP_ID` + `GITHUB_PRIVATE_KEY`. Creates per-installation clients with short-lived tokens. Git clone/push uses `x-access-token:{token}@github.com` URLs.

## Dependencies

- **claude** CLI — The agent command; must be installed and configured.
- **git** — Used by agent for clone, branch, commit, push.
- **github.com/jferrl/go-githubauth** — GitHub App JWT + installation token auth.
- **github.com/gin-gonic/gin** — HTTP router.
- **go.uber.org/zap** — Structured logging.

## Key Implementation Details

- **ClientAPI interface**: All GitHub operations go through `github.ClientAPI`. One implementation: `Client` (GitHub App installation tokens).
- **Webhook validation**: HMAC-SHA256 signature checked in `middleware.ValidateSignature()` before the handler sees the payload. Parsing via `gh.ParseWebHook` from go-github.
- **Duplicate prevention**: Webhook handler skips issues that already have the `llm-in-progress` label.
- **Label state machine**: `feature-request` → (manual) → `approved-for-dev` → `llm-in-progress` → `llm-pr-created` or `llm-failed`.
- **Feature-request guard**: `approved-for-dev` label is only processed if the issue also has `feature-request`.
- **Agent workspace**: `{workspace_dir}/{owner}/{repo}/repo` — reused across runs; `git fetch --all --prune` before new work.
- **Branch naming**: `vote-llm/issue-{number}-{slugified-title}` (slug max 40 chars).
- **Claude invocation**: `claude -p {prompt} --output-format json --allowedTools {tools} --max-turns 25 --max-budget-usd 5.00 --no-session-persistence`
- **Existing PR handling**: If PR creation fails, `FindPullRequestByHead` looks up an existing open PR for the branch.
- **Default branch**: Uses `GetDefaultBranch()` to discover the repo's default branch (not hardcoded to `main`).
- **Logging**: Structured logging via `go.uber.org/zap` with colored console output and component hierarchy (e.g. `webhook`, `agent`).
