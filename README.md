# github-vote-llm

A community-driven roadmap platform with integrated AI-powered feature implementation. Users submit and vote on feature proposals, and approved proposals can be automatically implemented by an AI agent (Claude Code) that opens a pull request.

## How it works

1. Users submit feature proposals via the community board (public, no login required)
2. The community votes on proposals
3. Maintainers review the roadmap and promote proposals through statuses (proposed → planned → in progress → done)
4. When a proposal is approved, the AI agent clones the target repo, runs Claude Code against the proposal, and opens a PR

## Project structure

```
client/                     React SPA (Vite + TypeScript + Tailwind + shadcn/ui)
  src/
    pages/                  LoginPage, SettingsPage, OrganizationDashboardPage, CreateOrganizationPage,
                            RepositoriesPage, RepositoryDetailPage, portal/
    components/             Layout, roadmap/ (React Flow canvas), shadcn/ui primitives
    lib/
      api.ts                API client (fetch + Bearer JWT + Zod validation)
      api-schemas.ts        Zod schemas for API responses
      auth-schemas.ts       Zod schemas for auth responses
      auth.tsx              OAuth2 PKCE auth context + token management
      portal-api.ts         API client for public portal
      pkce.ts               PKCE code challenge/verifier helpers
    hooks/
      use-mobile.ts         Mobile breakpoint hook
      use-portal-sse.ts     SSE subscription hook for portal events
    portal.tsx              Separate entrypoint for the public community board

server/                     Go backend (Gin + pgx/v5 + sqlc + River)
  cmd/main/main.go          Entry point: env config, DB connection, job client, Gin router
  db/
    migrations/             SQL migration files (apply with migrate CLI)
    queries/                sqlc query definitions
    sqlc.yaml               sqlc config
  internal/
    agents/
      agents.go             Runner interface
      claude/claude.go      ClaudeRunner: spawns `claude -p` CLI with streaming output
    api/
      api.go                Router setup (Gin)
      handlers/             handlers.go (factory), auth, users, organizations, github,
                            repositories, members, features, runs, portal
      middleware/            JWT auth, request ID, request logging, CORS
      request/              Context helpers (request ID extraction)
    config/                 Env var parsing (caarlos0/env) + token TTL constants
    dtos/                   Request/response types (auth, users, orgs, repos, runs, portal)
    encryption/             AES-256-GCM token encryption/decryption
    errors/                 Shared error helpers
    github/                 GitHub OAuth2 config + per-user token source (auto-refresh)
    helpers/                Shared utilities (password hashing, numeric conversion)
    hub/                    In-memory pub/sub for real-time SSE events
    jobs/
      client.go             River job client setup (PostgreSQL-backed queue)
      args/                 Job argument types (CloneRepoArgs, RunAgentArgs)
      workers/              workers.go (registry), clonerepo.go, runagent.go
    logger/                 Structured logging with colored output (zap)
    services/               services.go (factory), auth, users, organizations, github,
                            repositories, members, features, runs, portal
    store/                  PostgreSQL store (pgx/v5 + sqlc generated code)
  Makefile
```

## Requirements

- Go 1.24+
- Node.js 20+ with pnpm
- PostgreSQL 14+
- A [GitHub OAuth App](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app) for connecting user accounts
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and available on `$PATH`
- [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for running database migrations
- [air](https://github.com/air-verse/air) for live reload in development

## Configuration

All config is via environment variables (loaded from `.env` in debug mode):

| Variable               | Required | Description                                                                      |
| ---------------------- | -------- | -------------------------------------------------------------------------------- |
| `GITHUB_CLIENT_ID`     | yes      | GitHub OAuth App client ID                                                       |
| `GITHUB_CLIENT_SECRET` | yes      | GitHub OAuth App client secret                                                   |
| `FRONTEND_URL`         | yes      | Frontend base URL, used for post-OAuth redirect (e.g. `http://localhost:5173`)   |
| `SERVER_URL`           | yes      | Server base URL, used as the OAuth callback base (e.g. `http://localhost:8080`)  |
| `TOKEN_ENCRYPTION_KEY` | yes      | 64 hex chars (32 bytes) for AES-256-GCM encryption of stored GitHub tokens      |
| `WEBHOOK_SECRET`       | yes      | HMAC secret (for future webhook support)                                         |
| `ANTHROPIC_API_KEY`    | yes      | API key passed to the `claude` CLI                                               |
| `API_KEY`              | yes      | API key for `X-Api-Key` protected endpoints                                      |
| `DATABASE_URL`         | yes      | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/dbname`) |
| `JWT_SECRET`           | yes      | Secret for signing JWT access tokens                                             |
| `PORT`                 | no       | HTTP listen port (default: `8080`)                                               |
| `WORKSPACE_DIR`        | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`)                   |

## Database

```bash
# Install the migrate CLI (once)
go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply all migrations
cd server
make migrate-up

# Roll back the last migration
make migrate-down
```

### Tables

| Table                       | Purpose                                                             |
| --------------------------- | ------------------------------------------------------------------- |
| `users`                     | User accounts (email + bcrypt password)                             |
| `authorization_codes`       | OAuth2 authorization codes (PKCE, single-use, short-lived)          |
| `refresh_tokens`            | Long-lived refresh tokens (stored as SHA-256 hash)                  |
| `organizations`             | Tenant organizations                                                |
| `organization_members`      | Many-to-many: users in organizations with roles (owner, member)     |
| `repositories`              | Repos linked to organizations (owner/repo, portal settings)         |
| `github_connections`        | Encrypted GitHub OAuth tokens per user (AES-256-GCM + base64)      |
| `features`                  | Feature proposals with review/build status, votes, position         |
| `feature_comments`          | Comments on features                                                |
| `feature_votes`             | Vote tracking per feature per user                                  |
| `feature_dependencies`      | Dependency edges between features                                   |
| `feature_runs`              | Agent execution runs — status, prompt, workspace, timestamps        |
| `river_*`                   | River job queue tables (jobs, clients, queues, leaders)              |

## Agent execution pipeline

When a run is created (`POST /v1/features/:featureId/runs`), the system chains two background jobs via the River job queue:

1. **CloneRepoWorker** — clones the target repository into a workspace directory using an authenticated GitHub URL (skips if already cloned)
2. **RunAgentWorker** — creates a git worktree, runs `claude -p "<prompt>" --verbose` inside it (30-minute timeout), and updates the run status

Workspace layout: `{WORKSPACE_DIR}/{orgID}/{repoID}/` for the clone, with worktrees in `worktrees/run-{runID}/`. Each run gets its own branch: `feature-{featureID}-run-{runID}`.

## Running

```bash
# Server
cd server
go build -o vote-llm ./cmd/main
export DATABASE_URL=postgres://user:pass@localhost:5432/vote_llm
export JWT_SECRET=your-jwt-secret
# ... other required env vars
./vote-llm

# Client (dev)
cd client
pnpm install
pnpm dev

# Server (dev with live reload)
cd server
make dev    # starts air
```

## API

All endpoints are prefixed with `/v1`.

### Health

| Method | Path         | Auth | Description  |
| ------ | ------------ | ---- | ------------ |
| `GET`  | `/v1/health` | none | Health check |

### Auth (OAuth2 Authorization Code + PKCE)

| Method | Path                 | Auth | Description                                      |
| ------ | -------------------- | ---- | ------------------------------------------------ |
| `POST` | `/v1/auth/authorize` | none | Validate credentials, return authorization code  |
| `POST` | `/v1/auth/token`     | none | Exchange code or refresh token for access token  |
| `POST` | `/v1/auth/revoke`    | none | Revoke a refresh token                           |

### Users

| Method   | Path                     | Auth   | Description      |
| -------- | ------------------------ | ------ | ---------------- |
| `POST`   | `/v1/users/signup`       | none   | Create account   |
| `GET`    | `/v1/users/me`           | Bearer | Get profile      |
| `PATCH`  | `/v1/users/me/username`  | Bearer | Update username  |
| `DELETE` | `/v1/users/:id`          | Bearer | Delete account   |

### GitHub (connect GitHub account)

| Method   | Path                        | Auth   | Description                                    |
| -------- | --------------------------- | ------ | ---------------------------------------------- |
| `GET`    | `/v1/github/callback`       | none   | OAuth callback — GitHub redirects here         |
| `GET`    | `/v1/github/authorize`      | Bearer | Get GitHub OAuth authorization URL             |
| `GET`    | `/v1/github/status`         | Bearer | Check if GitHub is connected                   |
| `GET`    | `/v1/github/repositories`   | Bearer | List authenticated user's GitHub repos         |
| `DELETE` | `/v1/github/connection`     | Bearer | Disconnect GitHub account                      |

### Organizations

| Method   | Path                          | Auth   | Description           |
| -------- | ----------------------------- | ------ | --------------------- |
| `GET`    | `/v1/organizations`           | Bearer | List my organizations |
| `POST`   | `/v1/organizations`           | Bearer | Create organization   |
| `GET`    | `/v1/organizations/:id`       | Bearer | Get organization      |
| `PUT`    | `/v1/organizations/:id`       | Bearer | Update organization   |
| `PATCH`  | `/v1/organizations/:id/slug`  | Bearer | Update slug           |
| `DELETE` | `/v1/organizations/:id`       | Bearer | Delete organization   |

### Organization repositories

| Method   | Path                                              | Auth   | Description                          |
| -------- | ------------------------------------------------- | ------ | ------------------------------------ |
| `GET`    | `/v1/organizations/:id/repositories`              | Bearer | List repos connected to org          |
| `POST`   | `/v1/organizations/:id/repositories`              | Bearer | Add repo                             |
| `DELETE` | `/v1/organizations/:id/repositories/:repoId`      | Bearer | Remove repo from org                 |

### Organization members

| Method   | Path                                        | Auth   | Description                        |
| -------- | ------------------------------------------- | ------ | ---------------------------------- |
| `GET`    | `/v1/organizations/:id/members`             | Bearer | List members                       |
| `POST`   | `/v1/organizations/:id/members`             | Bearer | Invite by email                    |
| `DELETE` | `/v1/organizations/:id/members/:user_id`    | Bearer | Remove member                      |
| `PATCH`  | `/v1/organizations/:id/members/:user_id`    | Bearer | Update role                        |

### Repository features

| Method   | Path                                                                       | Auth   | Description         |
| -------- | -------------------------------------------------------------------------- | ------ | ------------------- |
| `GET`    | `/v1/repositories/:repoId/roadmap`                                         | Bearer | Get roadmap         |
| `GET`    | `/v1/repositories/:repoId/meta`                                            | Bearer | Get repo metadata   |
| `GET`    | `/v1/repositories/:repoId/features`                                        | Bearer | List features       |
| `POST`   | `/v1/repositories/:repoId/features`                                        | Bearer | Create feature      |
| `GET`    | `/v1/repositories/:repoId/features/:featureId`                             | Bearer | Get feature         |
| `PATCH`  | `/v1/repositories/:repoId/features/:featureId`                             | Bearer | Update feature      |
| `DELETE` | `/v1/repositories/:repoId/features/:featureId`                             | Bearer | Delete feature      |
| `PATCH`  | `/v1/repositories/:repoId/features/:featureId/position`                    | Bearer | Update position     |
| `GET`    | `/v1/repositories/:repoId/features/:featureId/comments`                    | Bearer | List comments       |
| `POST`   | `/v1/repositories/:repoId/features/:featureId/comments`                    | Bearer | Create comment      |
| `POST`   | `/v1/repositories/:repoId/features/:featureId/vote`                        | Bearer | Toggle vote         |
| `POST`   | `/v1/repositories/:repoId/features/:featureId/dependencies`                | Bearer | Add dependency      |
| `DELETE` | `/v1/repositories/:repoId/features/:featureId/dependencies/:dependsOn`     | Bearer | Remove dependency   |
| `PATCH`  | `/v1/repositories/:repoId/portal`                                          | Bearer | Update portal visibility |

### Feature runs

| Method | Path                               | Auth   | Description                          |
| ------ | ---------------------------------- | ------ | ------------------------------------ |
| `POST` | `/v1/features/:featureId/runs`     | Bearer | Create run (triggers agent pipeline) |

### Public portal

| Method | Path                                                              | Auth | Description       |
| ------ | ----------------------------------------------------------------- | ---- | ----------------- |
| `GET`  | `/v1/portal/:orgSlug/:repoName`                                   | none | Get portal page   |
| `GET`  | `/v1/portal/:orgSlug/:repoName/events`                            | none | SSE event stream  |
| `POST` | `/v1/portal/:orgSlug/:repoName/features/:featureId/vote`          | none | Toggle vote       |
| `GET`  | `/v1/portal/:orgSlug/:repoName/features/:featureId/comments`      | none | List comments     |
| `POST` | `/v1/portal/:orgSlug/:repoName/features/:featureId/comments`      | none | Create comment    |

## Auth flow

1. Client generates a PKCE code verifier + SHA-256 challenge
2. `POST /v1/auth/authorize` with email, password, code_challenge, redirect_uri → returns `code`
3. `POST /v1/auth/token` with `grant_type=authorization_code`, code, code_verifier → returns `access_token` (JWT, short-lived) + `refresh_token` (opaque, long-lived)
4. Protected requests include `Authorization: Bearer <access_token>`
5. On 401, client calls `POST /v1/auth/token` with `grant_type=refresh_token` to get a new access token
6. `POST /v1/auth/revoke` invalidates the refresh token on logout

## GitHub OAuth flow

1. Client calls `GET /v1/github/authorize` (Bearer) → server returns `{authorize_url}`
2. Client redirects user to `authorize_url` (GitHub)
3. User approves → GitHub redirects to `GET /v1/github/callback?code=...&state=...`
4. Server validates the signed JWT state, exchanges the code for a GitHub token
5. Token is AES-256-GCM encrypted and base64-encoded before storing in `github_connections`
6. Server redirects to `FRONTEND_URL?github_connected=1`
7. Subsequent GitHub API calls use `GithubTokenSource` which decrypts, returns valid tokens, and auto-refreshes when expired

## Development

```bash
# Server: regenerate store from SQL queries
cd server
make generate

# Server: create a new migration
make migrate-new name=add_something

# Client: install dependencies
cd client
pnpm install
```

## License

See [LICENSE](LICENSE) for details.
