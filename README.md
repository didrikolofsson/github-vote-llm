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
    pages/                  LoginPage, RoadmapPage, RunsPage, RunDetailPage, ConfigPage, BoardPage
    components/             Layout, StatusBadge, shadcn/ui primitives
    lib/
      api.ts                API client (fetch + Bearer JWT + Zod validation)
      api-schemas.ts        Zod schemas for API responses
      auth-schemas.ts       Zod schemas for auth responses
      auth.tsx              OAuth2 PKCE auth context + token management
      pkce.ts               PKCE code challenge/verifier helpers
    board.tsx               Separate entrypoint for the public community board

server/                     Go backend (Gin + pgx/v5 + sqlc)
  cmd/main/main.go          Entry point: env config, DB connection, Gin router
  db/
    migrations/             SQL migration files (apply with migrate CLI)
    queries/                sqlc query definitions
    sqlc.yaml               sqlc config
  internal/
    api/
      api.go                Router setup (Gin)
      handlers/             auth.go, users.go, organizations.go
      services/             auth.go, users.go, organizations.go
      dtos/                 auth.go, users.go, organizations.go
      middleware/           JWT auth, API key validation, request ID, request logging
      request/              Context helpers (request ID extraction)
    config/                 Env var parsing (caarlos0/env) + token TTL constants
    github/                 GitHub OAuth client for listing repos (go-github)
    helpers/                Shared utilities (password hashing, numeric conversion)
    logger/                 Structured logging with colored output (zap)
    spinner/                Terminal progress spinner
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

| Variable                 | Required | Description                                                                      |
| ------------------------ | -------- | -------------------------------------------------------------------------------- |
| `GITHUB_CLIENT_ID`       | yes      | GitHub OAuth App client ID                                                      |
| `GITHUB_CLIENT_SECRET`   | yes      | GitHub OAuth App client secret                                                  |
| `FRONTEND_URL`           | yes      | Frontend base URL for OAuth redirect (e.g. `http://localhost:5173`)             |
| `TOKEN_ENCRYPTION_KEY`   | yes      | 32-byte hex key for encrypting GitHub tokens at rest (64 hex chars)             |
| `WEBHOOK_SECRET`         | yes      | HMAC secret (legacy, for future webhook support)                                |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI                                               |
| `API_KEY`            | yes      | API key for REST endpoints (sent as `X-Api-Key` header)                          |
| `DATABASE_URL`       | yes      | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/dbname`) |
| `JWT_SECRET`         | yes      | Secret for signing JWT access tokens                                             |
| `PORT`               | no       | HTTP listen port (default: `8080`)                                               |
| `WORKSPACE_DIR`      | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`)                   |

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

| Table                   | Purpose                                                            |
| ----------------------- | ------------------------------------------------------------------ |
| `users`                 | User accounts (email + bcrypt password)                            |
| `authorization_codes`   | OAuth2 authorization codes (PKCE, single-use, short-lived)         |
| `refresh_tokens`        | Long-lived refresh tokens (stored as SHA-256 hash)                 |
| `organizations`         | Tenant organizations                                               |
| `organization_members`  | Many-to-many: users in organizations with roles (owner, member)    |
| `organization_repositories` | Junction: orgs link to repos (owner/repo)                       |
| `github_connections`   | Encrypted GitHub OAuth tokens per user                            |
| `proposals`             | Community feature proposals with vote counts and status            |
| `proposal_comments`     | Comments on proposals                                              |
| `repo_config`           | Per-repo overrides for labels, timeout, budget, and API key        |
| `executions`            | Tracks every agent run — status, branch, PR URL, error, timestamps |
| `issue_votes`           | Vote tracking per GitHub issue                                     |

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
make dev    # starts ngrok + air
```

## API

All endpoints are prefixed with `/v1`.

### Health

| Method | Path         | Auth | Description  |
| ------ | ------------ | ---- | ------------ |
| `GET`  | `/v1/health` | none | Health check |

### Auth (OAuth2 Authorization Code + PKCE)

| Method | Path               | Auth | Description                                      |
| ------ | ------------------ | ---- | ------------------------------------------------ |
| `POST` | `/v1/auth/authorize` | none | Validate credentials, return authorization code |
| `POST` | `/v1/auth/token`     | none | Exchange code or refresh token for access token  |
| `POST` | `/v1/auth/revoke`    | none | Revoke a refresh token                           |

### Users

| Method   | Path                | Auth   | Description     |
| -------- | ------------------- | ------ | --------------- |
| `POST`   | `/v1/users/signup`  | none   | Create account  |
| `DELETE` | `/v1/users/:id`     | Bearer | Delete account  |

### Organizations

| Method   | Path                      | Auth | Description              |
| -------- | ------------------------- | ---- | ------------------------ |
| `GET`    | `/v1/organizations`       | Bearer | List my organizations |
| `POST`   | `/v1/organizations/`      | Bearer | Create organization   |
| `GET`    | `/v1/organizations/:id`   | Bearer | Get organization      |
| `PUT`    | `/v1/organizations/:id`   | Bearer | Update organization   |
| `DELETE` | `/v1/organizations/:id`   | Bearer | Delete organization   |

### GitHub OAuth (Connect GitHub account)

| Method | Path                         | Auth   | Description                            |
| ------ | ---------------------------- | ------ | -------------------------------------- |
| `GET`  | `/v1/auth/github/status`    | Bearer | Check if GitHub is connected           |
| `GET`  | `/v1/auth/github/authorize`  | Bearer | Get GitHub OAuth URL (redirect user)   |
| `GET`  | `/v1/auth/github/callback`   | none   | OAuth callback (GitHub redirects here)  |

### Organization repositories

| Method | Path                                    | Auth   | Description                    |
| ------ | --------------------------------------- | ------ | ------------------------------ |
| `GET`  | `/v1/organizations/:id/repositories`    | Bearer | List org repositories          |
| `GET`  | `/v1/organizations/:id/repositories/available` | Bearer | List repos from GitHub (for adding) |
| `POST` | `/v1/organizations/:id/repositories`   | Bearer | Add repository (body: `{owner, repo}`) |
| `DELETE` | `/v1/organizations/:id/repositories/:owner/:repo` | Bearer | Remove repository     |

### Organization members

| Method | Path                              | Auth   | Description                    |
| ------ | --------------------------------- | ------ | ------------------------------ |
| `GET`  | `/v1/organizations/:id/members`   | Bearer | List members (with email)      |
| `POST` | `/v1/organizations/:id/members`   | Bearer | Invite by email (body: `{email}`) |
| `DELETE` | `/v1/organizations/:id/members/:user_id` | Bearer | Remove member          |
| `PATCH` | `/v1/organizations/:id/members/:user_id` | Bearer | Update role (body: `{role}`)    |

## Auth flow

1. Client generates a PKCE code verifier + SHA-256 challenge
2. `POST /v1/auth/authorize` with email, password, code_challenge, redirect_uri → returns `code`
3. `POST /v1/auth/token` with `grant_type=authorization_code`, code, code_verifier → returns `access_token` (JWT, short-lived) + `refresh_token` (opaque, long-lived)
4. Protected requests include `Authorization: Bearer <access_token>`
5. On 401, client calls `POST /v1/auth/token` with `grant_type=refresh_token` to get a new access token
6. `POST /v1/auth/revoke` invalidates the refresh token on logout

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
