# github-vote-llm

A Go service that automates feature implementation for GitHub repos. When an issue receives an approval label, it runs **Claude Code** (`claude` CLI) to implement the feature, commit changes, and open a PR.

## How It Works

1. **Webhook** — Listens for GitHub `issues` webhook events (labeled).
2. **Approval trigger** — When an issue gets the configured "approved" label (e.g. `approved-for-dev`), the agent is triggered.
3. **Agent flow** — For each approved issue:
   - Adds `llm-in-progress` label
   - Clones/updates the repo into a workspace
   - Creates a branch `vote-llm/issue-{n}-{slug}`
   - Runs `claude -p` with the issue as the prompt
   - Commits, pushes, and creates a PR
   - Adds `llm-pr-created` label, removes in-progress, comments with PR link

## Project Structure

```
cmd/server/main.go     # Entry point: flags, config, HTTP server, webhook forward (dev mode)
internal/
  agent/runner.go      # Orchestrates Claude Code: clone, branch, run claude, commit, PR
  cli/flag.go          # CLI flags (--config, --dev, --owner, --repo)
  config/config.go     # YAML config loading, env var expansion, validation
  github/client.go     # GitHub API: issues, labels, comments, PRs, webhooks
  github/webhook.go    # Webhook handler: validates payload, handles issues/labeled events
  votes/tracker.go     # Vote counting (+1 reactions) — NOT currently wired into main flow
```

## Configuration

`config.yaml`:

- **github.token**, **github.webhook_secret** — Env vars: `${GITHUB_TOKEN}`, `${WEBHOOK_SECRET}`
- **repos** — Per-repo: owner, name, labels (feature_request, approved, in_progress, done), vote_threshold
- **agent** — `command` (e.g. `claude`), max_turns, max_budget_usd, allowed_tools, workspace_dir

## Dev Mode

For local development without a real webhook:

```bash
go run ./cmd/server --dev --owner=didrikolofsson --repo=github-vote-llm
```

- Starts `gh webhook forward` as a subprocess to forward GitHub webhooks to `localhost`
- **Before** creating a new forward, removes existing `gh webhook forward` webhooks (avoids "Hook already exists" when a previous run was killed)
- On shutdown (SIGINT/SIGTERM), sends SIGTERM to the `gh` process and shuts down the server

## Dependencies

- **gh** CLI — Required for dev mode (`gh webhook forward`). Must be logged in.
- **claude** CLI — The agent command; must be installed and configured.
- **git** — Used by agent for clone, branch, commit, push.

## Key Implementation Details

- **Webhook validation**: Uses `gh.ValidatePayload` and `gh.ParseWebHook` from go-github.
- **Agent workspace**: `{workspace_dir}/{owner}/{repo}/repo` — reused across runs; `git fetch` before new work.
- **Branch naming**: `vote-llm/issue-{number}-{slugified-title}` (slug max 40 chars).
- **github/client.RemoveLocalRepoWebhooks**: Deletes hooks with URL `https://webhook-forwarder.github.com/hook` (gh webhook forward); used to clean up before creating a new forward.

## Unused Code

`internal/votes/tracker.go` — Vote tracker that counts +1 reactions and can add `votes:N+` labels. Not used in main.go. The current flow uses manual approval via labels, not vote thresholds.
