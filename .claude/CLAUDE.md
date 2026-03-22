# github-vote-llm

A community-driven roadmap platform with integrated AI-powered feature implementation. Users submit and vote on feature proposals via a public board; maintainers manage the roadmap; approved proposals can be automatically implemented by an AI agent (Claude Code) that opens a PR.

## Monorepo Layout

```
client/   React SPA (Vite + TypeScript + Tailwind + shadcn/ui)
server/   Go backend (Gin + pgx/v5 + sqlc)
```

## Server Structure

```
server/
  cmd/main/main.go                          Entry point: env config, DB, router wiring
  db/
    migrations/                             SQL migrations (000001–000007, apply with golang-migrate)
    queries/                                sqlc query definitions
    sqlc.yaml                               sqlc config
  internal/
    api/
      api.go                                Gin router setup; active routes + commented-out legacy routes
      handlers/auth.go                      OAuth2 handlers: Authorize, Token, Revoke
      handlers/users.go                     User handlers: SignupUser, DeleteUser
      handlers/organizations.go             Org handlers: Create, Get, Update, Delete
      services/auth.go                      AuthService: PKCE validation, JWT issuance, refresh/revoke
      services/users.go                     UserService: create, delete
      services/organizations.go             OrgService: CRUD with name-uniqueness guard
      dtos/auth.go                          JWT Claims struct
      dtos/users.go                         User request/response types
      dtos/organizations.go                 Org request/response types
      middleware/middleware.go              ValidateAPIKey, RequireAuth (JWT Bearer), AddRequestID, LogRequests
      request/request.go                   GetRequestID helper
    config/
      config.go                             Token TTL constants (AccessTokenTTL, RefreshTokenTTL, AuthCodeTTL)
      environment.go                        Env var struct parsed via caarlos0/env
    github/
      auth.go                               GitHub App auth: ClientFactory + installation tokens (go-githubauth)
      client.go                             ClientAPI interface + App-based Client implementation
    helpers/helpers.go                      VerifyPassword, float64↔pgtype.Numeric conversion
    logger/logger.go                        Structured logging: zap with colored console output
    spinner/spinner.go                      Terminal progress spinner
    store/db.go                             sqlc-generated Queries struct + New()
    store/types.go                          All model types (sqlc-generated): User, Organization, Proposal, Execution, etc.
    store/*.sql.go                          sqlc-generated query implementations
  Makefile
```

## Client Structure

```
client/src/
  App.tsx                                   Router: auth guard → LoginPage or main layout
  board.tsx                                 Separate Vite entrypoint for the public community board
  pages/
    LoginPage.tsx                           Email/password login (triggers OAuth2 PKCE flow)
    RoadmapPage.tsx                         Kanban-style roadmap by proposal status (protected)
    RunsPage.tsx                            List of agent runs (protected)
    RunDetailPage.tsx                       Single run detail (protected)
    ConfigPage.tsx                          Repo config management + board toggle (protected)
    BoardPage.tsx                           Public community proposal board
    DashboardPage.tsx                       Dashboard overview (protected)
  components/
    Layout.tsx                              App shell with nav
    StatusBadge.tsx                         Status pill component
    ui/                                     shadcn/ui primitives (button, dialog, input, label, switch, textarea)
  lib/
    api.ts                                  API client: fetch + Bearer JWT + auto-refresh on 401 + Zod validation
    api-schemas.ts                          Zod schemas: Run, RepoConfig, Proposal, ProposalComment
    auth-schemas.ts                         Zod schemas: AuthorizeResponse, TokenResponse, SignupResponse
    auth.tsx                                React context: OAuth2 PKCE login/logout, token storage, auto-refresh
    pkce.ts                                 PKCE: generateVerifier, generateChallenge (SHA-256 + base64url)
    utils.ts                                cn() utility (clsx + tailwind-merge)
```

## Active API Routes

All prefixed with `/v1`.

```
GET  /v1/health

POST /v1/auth/authorize      email+password+code_challenge+redirect_uri → auth code
POST /v1/auth/token          authorization_code or refresh_token → access_token + refresh_token
POST /v1/auth/revoke         revoke refresh token

POST   /v1/users/signup      create account (no auth)
DELETE /v1/users/:id         delete account (RequireAuth)

POST   /v1/organizations/    create org
GET    /v1/organizations/:id get org
PUT    /v1/organizations/:id update org
DELETE /v1/organizations/:id delete org
```

> The legacy runs/repos/roadmap/board routes are commented out in `api.go` during active refactoring.

## Auth Flow (OAuth2 Authorization Code + PKCE)

1. Client generates `code_verifier` (random) and `code_challenge` (SHA-256 of verifier, base64url)
2. `POST /v1/auth/authorize` validates credentials, stores auth code with challenge in DB
3. `POST /v1/auth/token` verifies PKCE, marks code used, returns JWT access token + opaque refresh token
4. API calls use `Authorization: Bearer <access_token>`
5. On 401, client retries after calling `/v1/auth/token` with `grant_type=refresh_token`
6. Logout calls `/v1/auth/revoke` to delete the refresh token from DB

Access tokens are short-lived JWTs (HS256). Refresh tokens are stored as SHA-256 hashes.

## Configuration

All via environment variables. `godotenv` loads `.env` when `GIN_MODE=debug`.

| Variable             | Required | Description                                                                      |
| -------------------- | -------- | -------------------------------------------------------------------------------- |
| `GITHUB_APP_ID`      | yes      | GitHub App numeric ID                                                            |
| `GITHUB_PRIVATE_KEY` | yes      | PEM bytes as a string (also accepts `GITHUB_PRIVATE_KEY_PATH` in dev)           |
| `WEBHOOK_SECRET`     | yes      | HMAC secret for webhook signature validation                                     |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI                                               |
| `API_KEY`            | yes      | Legacy API key for `X-Api-Key` protected endpoints                               |
| `DATABASE_URL`       | yes      | PostgreSQL connection string                                                     |
| `JWT_SECRET`         | yes      | Secret for signing JWT access tokens                                             |
| `PORT`               | no       | HTTP listen port (default: `8080`)                                               |
| `WORKSPACE_DIR`      | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`)                   |

## Database

7 migrations in `server/db/migrations/`:

| Migration | Content                                               |
| --------- | ----------------------------------------------------- |
| 000001    | `executions` table                                    |
| 000002    | `repo_config` table                                   |
| 000003    | `issue_votes` table                                   |
| 000004    | `proposals` + `proposal_comments` + `is_board_public` on `repo_config` |
| 000005    | `users` table                                         |
| 000006    | `authorization_codes` + `refresh_tokens` tables       |
| 000007    | `organizations` + `organization_members` tables       |

Run `make migrate-up` from `server/` (requires `DATABASE_URL` in env).

## Key Implementation Details

- **sqlc**: all store queries are type-safe and generated. Run `make generate` to regenerate after changing `db/queries/*.sql`.
- **JWT claims**: `dtos.Claims` embeds `jwt.RegisteredClaims` and carries `UserID` + `Email`. Validated by `middleware.RequireAuth`.
- **PKCE**: server-side verification in `services.AuthService.ExchangeCode` — computes SHA-256 of verifier and compares to stored challenge.
- **Refresh token storage**: only the SHA-256 hash is stored in `refresh_tokens`; the raw token is sent to the client once and never stored.
- **Organization uniqueness**: `services.ErrOrganizationNameExists` is returned on name conflict; handler maps it to 400.
- **Request IDs**: `middleware.AddRequestID` sets a UUID per request via `c.Set("request_id", ...)`; `request.GetRequestID` retrieves it for logging.
- **GitHub App auth**: `github.ClientFactory` creates per-installation clients with short-lived tokens. Git operations use `x-access-token:{token}@github.com` URLs.
- **Agent flow** (being refactored, routes commented out): runs `claude -p` with issue prompt, commits + pushes, opens PR; tracks state in `executions` table.
- **Logging**: `go.uber.org/zap` with colored console output; named loggers per component (e.g., `logger.New().Named("api")`).
- **Module path**: `github.com/didrikolofsson/github-vote-llm`

## Server Makefile Targets

```
make dev            # ngrok + air (live reload)
make build          # go build ./...
make test           # go test ./...
make generate       # sqlc generate -f db/sqlc.yaml
make generate-pkce  # node scripts/pkce.js (test PKCE locally)
make migrate-new    # create new migration (name=<migration_name>)
make migrate-up     # apply all pending migrations
make migrate-down   # roll back last migration
```

## Dependencies

**Server:**
- `github.com/gin-gonic/gin` — HTTP router
- `github.com/jackc/pgx/v5` — PostgreSQL driver + connection pool
- `github.com/caarlos0/env/v11` — env var parsing
- `go.uber.org/zap` — structured logging
- `github.com/golang-jwt/jwt/v5` — JWT signing/verification
- `github.com/google/uuid` — request ID generation
- `github.com/jferrl/go-githubauth` — GitHub App JWT + installation tokens
- `github.com/joho/godotenv` — `.env` file loading in dev
- `sqlc` — SQL → Go code generation
- `golang-migrate` CLI — migration runner

**Client:**
- React + Vite + TypeScript
- React Router v6
- Tailwind CSS + shadcn/ui
- Zod — runtime schema validation
