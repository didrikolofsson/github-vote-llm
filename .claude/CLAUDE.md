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
    migrations/                             SQL migrations (000001–000008, apply with golang-migrate)
    queries/                                sqlc query definitions
    sqlc.yaml                               sqlc config
  internal/
    api/
      api.go                                Gin router setup
      handlers/auth.go                      OAuth2 handlers: Authorize, Token, Revoke
      handlers/users.go                     User handlers: SignupUser, DeleteUser
      handlers/organizations.go             Org handlers: ListMy, Create, Get, Update, Delete
      handlers/github.go                    GitHub handlers: Authorize, Callback, Status, ListReposByAuthenticatedUser
      handlers/repositories.go             Org repo handlers: List, Add, Remove
      handlers/members.go                   Org member handlers: List, Invite, Remove, UpdateRole
      services/auth.go                      AuthService: PKCE validation, JWT issuance, refresh/revoke
      services/users.go                     UserService: create, delete
      services/organizations.go             OrgService: CRUD with name-uniqueness guard
      services/github.go                    GithubService: OAuth callback, status check, list repos
      services/repositories.go             RepositoriesService: org repo CRUD
      services/members.go                   MembersService: org member management
      dtos/auth.go                          JWT Claims struct
      dtos/users.go                         User request/response types
      dtos/organizations.go                 Org request/response types
      dtos/github.go                        GitHubRepository, GitHubRepositoryListResponse
      middleware/middleware.go              ValidateAPIKey, RequireAuth (JWT Bearer), AddRequestID, LogRequests
      request/request.go                   GetRequestID helper
    config/
      config.go                             Token TTL constants (AccessTokenTTL, RefreshTokenTTL, AuthCodeTTL)
      environment.go                        Env var struct parsed via caarlos0/env
    encryption/encryption.go               AES-256-GCM Encrypt/Decrypt (key = 64 hex chars)
    oauth2/github.go                        NewGitHubOAuthConfig + GithubTokenSource (oauth2.TokenSource, auto-refresh)
    helpers/helpers.go                      VerifyPassword, float64↔pgtype.Numeric conversion
    logger/logger.go                        Structured logging: zap with colored console output
    spinner/spinner.go                      Terminal progress spinner
    store/db.go                             sqlc-generated Queries struct + New()
    store/types.go                          All model types (sqlc-generated)
    store/*.sql.go                          sqlc-generated query implementations
  Makefile
```

## Client Structure

```
client/src/
  App.tsx                                   Router: auth guard → LoginPage or main layout
  board.tsx                                 Separate Vite entrypoint for the public community board
  main.tsx                                  Main entrypoint
  pages/
    LoginPage.tsx                           Email/password login (triggers OAuth2 PKCE flow)
    SettingsPage.tsx                        GitHub connection status/connect + org members management
    OrganizationDashboardPage.tsx           Connected repos list + add/remove repos dialog
    CreateOrganizationPage.tsx              Create first organization
  components/
    Layout.tsx                              App shell with nav
    ui/                                     shadcn/ui primitives
  lib/
    api.ts                                  API client: fetch + Bearer JWT + auto-refresh on 401 + Zod validation
    api-schemas.ts                          Zod schemas: Run, RepoConfig, Proposal, ProposalComment
    auth-schemas.ts                         Zod schemas: AuthorizeResponse, TokenResponse, SignupResponse
    auth.tsx                                React context: OAuth2 PKCE login/logout, token storage, auto-refresh
    pkce.ts                                 PKCE: generateVerifier, generateChallenge (SHA-256 + base64url)
    utils.ts                                cn() utility (clsx + tailwind-merge)
    logger.ts                               Client-side logger
  hooks/
    use-mobile.ts
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

GET /v1/github/callback              public — GitHub redirects here after user approves
GET /v1/github/authorize             Bearer — returns { authorize_url }
GET /v1/github/status                Bearer — returns { connected, login? }
GET /v1/github/repositories          Bearer — lists authenticated user's GitHub repos (?page=N)

GET    /v1/organizations             list my organizations
POST   /v1/organizations             create org
GET    /v1/organizations/:id         get org
PUT    /v1/organizations/:id         update org
DELETE /v1/organizations/:id         delete org

GET    /v1/organizations/:id/repositories              list org repos
POST   /v1/organizations/:id/repositories              add repo { owner, repo }
DELETE /v1/organizations/:id/repositories/:owner/:repo remove repo

GET    /v1/organizations/:id/members                   list members
POST   /v1/organizations/:id/members                   invite member { email }
DELETE /v1/organizations/:id/members/:user_id          remove member
PATCH  /v1/organizations/:id/members/:user_id          update role { role }
```

## Auth Flow (OAuth2 Authorization Code + PKCE)

1. Client generates `code_verifier` (random) and `code_challenge` (SHA-256 of verifier, base64url)
2. `POST /v1/auth/authorize` validates credentials, stores auth code with challenge in DB
3. `POST /v1/auth/token` verifies PKCE, marks code used, returns JWT access token + opaque refresh token
4. API calls use `Authorization: Bearer <access_token>`
5. On 401, client retries after calling `/v1/auth/token` with `grant_type=refresh_token`
6. Logout calls `/v1/auth/revoke` to delete the refresh token from DB

Access tokens are short-lived JWTs (HS256). Refresh tokens are stored as SHA-256 hashes.

## GitHub OAuth Flow (connect GitHub account)

1. Client calls `GET /v1/github/authorize` (Bearer) → server returns `{ authorize_url }`
2. Client redirects user to GitHub
3. GitHub redirects to `GET /v1/github/callback?code=...&state=...`
4. Server validates state (signed JWT with userID + 10min expiry), exchanges code for GitHub token
5. Token is AES-256-GCM encrypted → base64-encoded → stored in `github_connections`
6. Server redirects to `FRONTEND_URL?github_connected=1`
7. `GET /v1/github/status` and `GET /v1/github/repositories` use `GithubTokenSource` which decrypts and auto-refreshes tokens

The two OAuth flows serve different roles: the app's own auth flow uses PKCE (app is the OAuth server); the GitHub connect flow uses the `golang.org/x/oauth2` client library (app is the OAuth client).

## Configuration

All via environment variables. `godotenv` loads `.env` when `GIN_MODE=debug`.

| Variable               | Required | Description                                                                      |
| ---------------------- | -------- | -------------------------------------------------------------------------------- |
| `GITHUB_CLIENT_ID`     | yes      | GitHub OAuth App client ID                                                       |
| `GITHUB_CLIENT_SECRET` | yes      | GitHub OAuth App client secret                                                   |
| `FRONTEND_URL`         | yes      | Frontend base URL for post-OAuth redirect (e.g. `http://localhost:5173`)         |
| `SERVER_URL`           | yes      | Server base URL used to build the OAuth callback (e.g. `http://localhost:8080`)  |
| `TOKEN_ENCRYPTION_KEY` | yes      | 64 hex chars (32 bytes) for AES-256-GCM encryption of stored GitHub tokens      |
| `WEBHOOK_SECRET`       | yes      | HMAC secret for future webhook support                                           |
| `ANTHROPIC_API_KEY`    | yes      | API key passed to the `claude` CLI                                               |
| `API_KEY`              | yes      | Legacy API key for `X-Api-Key` protected endpoints                               |
| `DATABASE_URL`         | yes      | PostgreSQL connection string                                                     |
| `JWT_SECRET`           | yes      | Secret for signing JWT access tokens                                             |
| `PORT`                 | no       | HTTP listen port (default: `8080`)                                               |
| `WORKSPACE_DIR`        | no       | Base dir for repo clones (default: `/tmp/vote-llm-workspaces`)                   |

## Database

8 migrations in `server/db/migrations/`:

| Migration | Content                                                                  |
| --------- | ------------------------------------------------------------------------ |
| 000001    | `executions` table                                                       |
| 000002    | `repo_config` table                                                      |
| 000003    | `issue_votes` table                                                      |
| 000004    | `proposals` + `proposal_comments` + `is_board_public` on `repo_config`  |
| 000005    | `users` table                                                            |
| 000006    | `authorization_codes` + `refresh_tokens` tables                          |
| 000007    | `organizations` + `organization_members` tables                          |
| 000008    | `organization_repositories` + `github_connections` tables; `repo_config` recreated |

Run `make migrate-up` from `server/` (requires `DATABASE_URL` in env).

## Key Implementation Details

- **sqlc**: all store queries are type-safe and generated. Run `make generate` to regenerate after changing `db/queries/*.sql`.
- **JWT claims**: `dtos.Claims` embeds `jwt.RegisteredClaims` and carries `UserID` + `Email`. Validated by `middleware.RequireAuth`.
- **PKCE**: server-side verification in `services.AuthService.ExchangeCode` — computes SHA-256 of verifier and compares to stored challenge.
- **Refresh token storage**: only the SHA-256 hash is stored in `refresh_tokens`; the raw token is sent to the client once and never stored.
- **GitHub token storage**: AES-256-GCM encrypted, then base64-encoded, stored as `TEXT` in `github_connections`. Both access and refresh tokens follow this pattern.
- **GithubTokenSource**: implements `oauth2.TokenSource`. On each call it decrypts the token from DB, returns it if valid, otherwise refreshes via `config.TokenSource`, re-encrypts, and upserts back to DB.
- **OAuth callback redirect_uri**: must match exactly what's registered in the GitHub OAuth App. The server builds it as `SERVER_URL + "/v1/github/callback"` in both the authorization URL and the token exchange.
- **Organization uniqueness**: `services.ErrOrganizationNameExists` is returned on name conflict; handler maps it to 400.
- **Request IDs**: `middleware.AddRequestID` sets a UUID per request via `c.Set("request_id", ...)`; `request.GetRequestID` retrieves it for logging.
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
- `github.com/google/go-github/v68` — GitHub API client
- `golang.org/x/oauth2` — OAuth2 client (GitHub token exchange + auto-refresh)
- `github.com/joho/godotenv` — `.env` file loading in dev
- `sqlc` — SQL → Go code generation
- `golang-migrate` CLI — migration runner

**Client:**
- React + Vite + TypeScript
- React Router v6
- TanStack Query (react-query)
- Tailwind CSS + shadcn/ui
- Zod — runtime schema validation
