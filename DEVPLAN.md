# Dev Plan

## Context

The current codebase is a minimal MVP:
- Webhook handler triggers the agent when `approved-for-dev` label is added to a `feature-request` issue
- Agent clones the repo, runs `claude -p`, commits, pushes, opens a PR
- GitHub App auth only (no PAT mode, no dev mode)
- No persistence (no database)
- No config system (all labels and agent settings hardcoded in `runner.go`)
- No vote tracking (referenced in CLAUDE.md but not implemented)

---

## 1. Blobless clone

**File:** `internal/agent/runner.go` → `cloneOrResetRepo`

Change `git clone` to use `--filter=blob:none`. Working tree is fully populated at checkout; only historical blobs from unreferenced branches are deferred. Safe for Claude — no files are missing.

```go
cmd := exec.CommandContext(ctx, "git", "clone", "--filter=blob:none", cloneURL, repoDir)
```

---

## 2. Persistence layer (PostgreSQL)

Add `internal/store` package backed by PostgreSQL. Using Postgres from the start avoids a SQLite→Postgres migration later and fits naturally with a deployed service (connection pooling, concurrent writes from multiple goroutines, hosted options like Supabase/RDS/Fly Postgres).

Connection string via env var `DATABASE_URL`. Use `pgx/v5` as the driver with `pgxpool` for connection pooling. Schema managed with sequential migration files (no ORM).

**Execution records** — idempotency and run status tracking:
```
id, owner, repo, issue_number, branch, status (pending|running|done|failed),
pr_url, error_message, created_at, updated_at
```

**Repo config** — per-repo settings editable via UI:
```
owner, repo, label_approved, label_feature_request, label_in_progress,
label_done, label_failed, vote_threshold, agent_timeout_minutes,
anthropic_api_key, updated_at
```

Wire the store into `WebhookHandler` and `Runner`. On label event: create execution record before launching goroutine (idempotency check replaces the current in-progress label guard).

---

## 3. Per-client API keys

Currently `claude` is invoked without an explicit API key, relying on the server's environment. This won't work for multi-client deployments.

- Store `anthropic_api_key` per repo in the config table (above)
- When running `claude`, inject `ANTHROPIC_API_KEY` into the subprocess env:

```go
cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+apiKey)
```

- UI config panel includes an API key field (write-only display, masked)
- `agentMaxBudgetUSD` remains as a server-side cap; actual spend is the client's responsibility

---

## 4. Sandbox cleanup

`gitCheckoutNewBranch` already runs `git clean -fd` before each run — this handles leftover files from the previous run on the same repo.

Ensure `implement()` cleans up on failure paths too: if the run fails after the workspace is set up, the branch is left behind. This is fine (it gets overwritten on retry via `-B` flag in `git checkout`), but the workspace directory itself (`/tmp/vote-llm-workspaces/{owner}/{repo}`) accumulates across repos. Add a deferred cleanup or a periodic pruning mechanism for workspaces of repos that are no longer active.

---

## 5. REST API

Add an `/api` route group to the existing Gin server, protected by a shared-secret middleware (header: `X-API-Key`).

Endpoints:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/runs` | List all execution records (filterable by repo/status) |
| `GET` | `/api/runs/:id` | Single run detail (status, error, PR URL, logs) |
| `POST` | `/api/runs/:id/retry` | Re-queue a failed run |
| `POST` | `/api/runs/:id/cancel` | Cancel an in-progress run |
| `GET` | `/api/repos` | List repos seen by the server |
| `GET` | `/api/repos/:owner/:repo/config` | Get repo config |
| `PUT` | `/api/repos/:owner/:repo/config` | Update repo config (labels, threshold, API key, timeout) |

The retry endpoint re-runs the agent for the same issue; cancel sets a context cancellation and updates the record status.

---

## 6. Minimal UI

A single-page frontend served by the Go server at `/ui`. Scope for the test cohort:

**Dashboard**
- List of runs across repos: repo, issue number + title, status badge, timestamp, link to PR

**Run detail**
- Status, error message (on failure), link to PR or branch, retry/cancel buttons

**Config panel**
- Per-repo: trigger label (`approved-for-dev`), vote threshold, timeout, API key (masked input)
- Changes POST to `/api/repos/:owner/:repo/config`

**Auth:** `X-API-Key` header via a login screen that stores the key in `localStorage`. No OAuth for now — upgrade path when the service becomes multi-user.

---

## 7. Vote tracking

Implement the vote counting logic referenced in CLAUDE.md but not yet built:
- Listen for `issue_comment` events and `feature-request` label additions in the webhook handler
- Count +1 reactions via `GetReactionCount`
- Add `candidate` label when threshold is met
- Threshold is read from per-repo config (store, not hardcoded)

This is a prerequisite for the UI voting panel to be meaningful.

---

## 8. Dev mode removal

Dev mode (`gh webhook forward` subprocess, `--dev` flag) is already absent from the actual codebase. Remove references from CLAUDE.md and ensure no dead code paths remain.

---

## Work order

```
1. Blobless clone          — small, self-contained, do first
2. PostgreSQL store        — unblocks everything else
3. Per-client API keys     — depends on store (config table)
4. Sandbox cleanup         — small, parallel with store
5. Vote tracking           — depends on store
6. REST API                — depends on store
7. Minimal UI              — depends on REST API
```
