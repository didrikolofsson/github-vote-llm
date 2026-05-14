# github-vote-llm

A community-driven roadmap platform with integrated AI-powered feature implementation. Users submit and vote on feature proposals via a public board; maintainers manage the roadmap; approved proposals can be automatically implemented by an AI agent (Claude Code) that opens a PR.

## Monorepo Layout

```
client/   React SPA (Vite + TypeScript + Tailwind + shadcn/ui)
server/   Go backend (Gin + pgx/v5 + sqlc + River job queue)
```

## Server Structure

```
server/
  cmd/main/main.go                          Entry point: env config, DB, job client, router wiring
  db/
    migrations/                             SQL migrations (000001–000007, apply with golang-migrate)
    queries/                                sqlc query definitions
    sqlc.yaml                               sqlc config
  internal/
    agents/
      agents.go                             Runner interface (Run(ctx, prompt) error)
      claude/claude.go                      ClaudeRunner: spawns `claude -p` CLI, streams stdout/stderr
    api/
      api.go                                Gin router setup + route registration
      handlers/
        handlers.go                         Handlers struct + New() factory wiring all handler groups
        auth.go                             OAuth2 handlers: Authorize, Token, Revoke
        users.go                            User handlers: SignupUser, GetMe, UpdateUsername, DeleteUser
        organizations.go                    Org handlers: ListMy, Create, Get, Update, UpdateSlug, Delete
        github.go                           GitHub handlers: Authorize, Callback, Status, ListRepos, Disconnect
        repositories.go                     Repo handlers: List, Add, Remove, GetRepoMeta, UpdatePortalVisibility
        members.go                          Org member handlers: List, Invite, Remove, UpdateRole
        features.go                         Feature handlers: List, Create, Get, Patch, Delete, roadmap, comments, votes, deps
        runs.go                             Run handlers: Create (triggers agent execution)
        portal.go                           Portal handlers: GetPortalPage, Subscribe (SSE), ToggleVote, comments
      middleware/
        middleware.go                       AddRequestID, LogRequests, RequireAuth (JWT Bearer)
        cors.go                             CORS configuration
      request/request.go                    GetRequestID helper
    config/
      config.go                             Token TTL constants (AccessTokenTTL, RefreshTokenTTL, AuthCodeTTL)
      environment.go                        Env var struct parsed via caarlos0/env
    dtos/                                   Data transfer objects (request/response types)
      auth.go                               JWT Claims struct
      users.go                              User request/response types
      organizations.go                      Org request/response types
      repositories.go                       Repository DTOs
      github.go                             GitHubRepository DTO
      runs.go                               RunDTO with status enum
      portal.go                             Portal DTOs
    encryption/encryption.go                AES-256-GCM Encrypt/Decrypt (key = 64 hex chars)
    errors/api_errors.go                    Shared error helpers (IsForeignKeyViolationErr, etc.)
    github/client.go                        NewGithubOAuthConfig + GithubTokenSource (oauth2.TokenSource, auto-refresh)
    helpers/helpers.go                      VerifyPassword, float64/pgtype.Numeric conversion
    hub/hub.go                              In-memory pub/sub for real-time SSE events (Subscribe/Publish per repoID)
    jobs/
      client.go                             River job client setup (NewClient with pgx driver)
      args/
        github.go                           CloneRepoArgs (Kind: "clone_repo")
        agents.go                           RunAgentArgs (Kind: "run_agent")
        pr.go                               OpenRepoPullRequestArgs (Kind: "open_pr", stub)
      workers/
        workers.go                          Register() — adds all workers to the river.Workers registry
        clonerepo.go                        CloneRepoWorker: clones repo into workspace, then enqueues RunAgentArgs
        runagent.go                         RunAgentWorker: prepares git worktree, runs Claude CLI (30min timeout)
    logger/logger.go                        Structured logging: zap with colored console output
    services/
      services.go                           ServicesDeps + Services struct + New() factory
      auth.go                               AuthService: PKCE validation, JWT issuance, refresh/revoke
      users.go                              UserService: create, get, update, delete
      organizations.go                      OrgService: CRUD with slug-uniqueness guard
      github.go                             GithubService: OAuth flow, clone repo to workspace, open PR
      repositories.go                       RepositoriesService: org repo CRUD
      members.go                            MembersService: org member management
      features.go                           FeaturesService: feature CRUD, votes, comments, roadmap, deps
      runs.go                               RunService: create run + dispatch clone job, run agent in worktree
      portal.go                             PortalService: public portal queries
    store/
      db.go                                 DBTX interface + Queries struct + New() + WithTx()
      types.go                              Enum types + model structs (sqlc-generated)
      *.sql.go                              Query implementations (sqlc-generated)
  Makefile
```

## Client Structure

```
client/src/
  App.tsx                                   Router: auth guard → LoginPage or main layout
  main.tsx                                  Main entrypoint
  portal.tsx                                Separate Vite entrypoint for the public community board
  pages/
    LoginPage.tsx                           Email/password login (triggers OAuth2 PKCE flow)
    SettingsPage.tsx                        GitHub connection status/connect + org members management
    OrganizationDashboardPage.tsx           Org overview
    CreateOrganizationPage.tsx              Create first organization
    RepositoriesPage.tsx                    Connected repos list + add/remove repos
    RepositoryDetailPage.tsx                Single repo view with features + roadmap
    portal/
      PortalPage.tsx                        Public portal for a repo
      ProposalsBoard.tsx                    Feature proposals board
      FeatureCard.tsx                       Feature card component
      FeatureSheet.tsx                      Feature detail sheet
      CommentForm.tsx                       Comment form
      RoadmapColumns.tsx                    Roadmap column view
      RecentlyShipped.tsx                   Shipped features section
  components/
    Layout.tsx                              App shell with nav
    roadmap/
      RoadmapCanvas.tsx                     React Flow canvas for roadmap visualization
      FeatureNode.tsx                       Custom node for features
      FeatureDrawer.tsx                     Feature detail drawer
    ui/                                     shadcn/ui primitives
  lib/
    api.ts                                  API client: fetch + Bearer JWT + auto-refresh on 401 + Zod validation
    api-schemas.ts                          Zod schemas for API responses
    auth-schemas.ts                         Zod schemas: AuthorizeResponse, TokenResponse, SignupResponse
    auth.tsx                                React context: OAuth2 PKCE login/logout, token storage, auto-refresh
    portal-api.ts                           API client for the public portal
    pkce.ts                                 PKCE: generateVerifier, generateChallenge (SHA-256 + base64url)
    utils.ts                                cn() utility (clsx + tailwind-merge)
    logger.ts                               Client-side logger
  hooks/
    use-mobile.ts                           Mobile breakpoint hook
    use-portal-sse.ts                       SSE subscription hook for portal events
```

## Adding Services, Handlers, and Jobs

This section describes the pattern for extending the backend with new services, handlers, and job workers. All three follow a consistent dependency-injection style wired through factory functions.

### 1. Adding a new service

Create the service file in `server/internal/services/`:

```go
// server/internal/services/billing.go
type BillingService struct {
    db *pgxpool.Pool
    q  *store.Queries
    jc *river.Client[pgx.Tx]   // only if the service needs to dispatch jobs
}

func NewBillingService(db *pgxpool.Pool, q *store.Queries, jc *river.Client[pgx.Tx]) *BillingService {
    return &BillingService{db: db, q: q, jc: jc}
}
```

Then register it in `services.go`:

```go
// Add to the Services struct
type Services struct {
    // ...existing fields...
    BillingService *BillingService
}

// Add to the New() factory
func New(deps ServicesDeps) *Services {
    return &Services{
        // ...existing services...
        BillingService: NewBillingService(deps.DB, deps.Queries, deps.JobClient),
    }
}
```

Services receive their dependencies via constructor params — `*pgxpool.Pool` for transactions, `*store.Queries` for database access, `*river.Client[pgx.Tx]` for dispatching jobs, `*config.Environment` for config, `hub.Hub` for events. Only inject what the service actually needs.

### 2. Adding a new handler

Create the handler file in `server/internal/api/handlers/`:

```go
// server/internal/api/handlers/billing.go
type BillingHandlers struct {
    s *services.BillingService
    l *logger.Logger
}

func NewBillingHandlers(s *services.BillingService, l *logger.Logger) *BillingHandlers {
    return &BillingHandlers{s: s, l: l}
}

func (h *BillingHandlers) GetInvoice(c *gin.Context) {
    // Parse params → call service → handle errors with errors.Is() → return JSON
}
```

Then register it in `handlers.go`:

```go
type Handlers struct {
    // ...existing fields...
    Billing *BillingHandlers
}

func New(deps NewHandlersDeps) Handlers {
    return Handlers{
        // ...existing handlers...
        Billing: NewBillingHandlers(deps.Services.BillingService, deps.Logger),
    }
}
```

Finally, add routes in `api.go`:

```go
billing := api.Group("/billing")
billing.Use(middleware.RequireAuth(env.JWT_SECRET))
billing.GET("/invoices", h.Billing.GetInvoice)
```

### 3. Adding a new job worker

Jobs use [River](https://riverqueue.com), a PostgreSQL-based job queue. Jobs are stored in `river_jobs` (migration 000006) and processed in-process by registered workers.

**Step 1 — Define args** in `server/internal/jobs/args/`:

```go
// server/internal/jobs/args/billing.go
type GenerateInvoiceArgs struct {
    OrgID int64 `json:"org_id"`
}

func (GenerateInvoiceArgs) Kind() string { return "generate_invoice" }
```

The `Kind()` string is the job type identifier — River uses it to route jobs to the correct worker.

**Step 2 — Create worker** in `server/internal/jobs/workers/`:

```go
// server/internal/jobs/workers/billing.go
type GenerateInvoiceWorker struct {
    river.WorkerDefaults[args.GenerateInvoiceArgs]
    svc *services.BillingService
}

func (w *GenerateInvoiceWorker) Work(ctx context.Context, job *river.Job[args.GenerateInvoiceArgs]) error {
    return w.svc.GenerateInvoice(ctx, job.Args.OrgID)
}
```

Override `Timeout()` if the job needs more than the default duration (see `RunAgentWorker` for a 30-minute example).

**Step 3 — Register** in `workers.go`:

```go
func Register(w *river.Workers, deps RegisterWorkersDeps) {
    // ...existing workers...
    river.AddWorker(w, &GenerateInvoiceWorker{svc: deps.Services.BillingService})
}
```

**Step 4 — Dispatch** from any service that has the `jc` (job client):

```go
// Simple insert (standalone)
s.jc.Insert(ctx, args.GenerateInvoiceArgs{OrgID: orgID}, nil)

// Insert within a transaction (atomically with other DB writes)
s.jc.InsertTx(ctx, tx, args.GenerateInvoiceArgs{OrgID: orgID}, nil)
```

### Existing job pipeline

The current agent execution pipeline chains two jobs:

```
POST /v1/features/:featureId/runs
  → RunService.CreateRun()
    → creates feature_run record (status: pending)
    → creates workspace directory
    → inserts CloneRepoArgs job

CloneRepoWorker.Work()
  → GithubService.CloneRepoToWorkspace()
    → clones repo with authenticated URL (skips if already cloned)
    → inserts RunAgentArgs job (next stage)

RunAgentWorker.Work() [30 min timeout]
  → RunService.RunAgent()
    → updates run status: pending → running
    → prepares git worktree (branch: feature-{id}-run-{id})
    → executes `claude -p "<prompt>" --verbose` in worktree
    → updates run status: → completed or → failed
```

### Adding a new database query

```sql
-- 1. Add SQL to server/db/queries/billing.sql
-- name: GetInvoice :one
SELECT * FROM invoices WHERE id = $1;
```

Then run `make generate` from `server/`. This generates `server/internal/store/billing.sql.go`. Use in services via `s.q.GetInvoice(ctx, id)`.

### Adding a database migration

```bash
cd server
make migrate-new name=add_invoices_table
# Edit the generated up/down files in db/migrations/
make migrate-up
make generate   # regenerate sqlc after schema changes
```

## Active API Routes

All prefixed with `/v1`.

```
GET  /v1/health

POST /v1/auth/authorize               email+password+code_challenge+redirect_uri → auth code
POST /v1/auth/token                    authorization_code or refresh_token → tokens
POST /v1/auth/revoke                   revoke refresh token

POST   /v1/users/signup                create account (no auth)
GET    /v1/users/me                    get profile (Bearer)
PATCH  /v1/users/me/username           update username (Bearer)
DELETE /v1/users/:id                   delete account (Bearer)

GET    /v1/github/callback             public — GitHub OAuth callback
GET    /v1/github/authorize            Bearer — returns { authorize_url }
GET    /v1/github/status               Bearer — returns { connected, login? }
GET    /v1/github/repositories         Bearer — lists authenticated user's GitHub repos (?page=N)
DELETE /v1/github/connection           Bearer — disconnect GitHub account

GET    /v1/organizations               list my organizations (Bearer)
POST   /v1/organizations               create org (Bearer)
GET    /v1/organizations/:id           get org (Bearer)
PUT    /v1/organizations/:id           update org (Bearer)
PATCH  /v1/organizations/:id/slug      update org slug (Bearer)
DELETE /v1/organizations/:id           delete org (Bearer)

GET    /v1/organizations/:id/repositories              list org repos (Bearer)
POST   /v1/organizations/:id/repositories              add repo (Bearer)
DELETE /v1/organizations/:id/repositories/:repoId      remove repo (Bearer)

GET    /v1/organizations/:id/members                   list members (Bearer)
POST   /v1/organizations/:id/members                   invite member { email } (Bearer)
DELETE /v1/organizations/:id/members/:user_id          remove member (Bearer)
PATCH  /v1/organizations/:id/members/:user_id          update role { role } (Bearer)

GET    /v1/repositories/:repoId/roadmap                          get roadmap (Bearer)
GET    /v1/repositories/:repoId/meta                             get repo metadata (Bearer)
GET    /v1/repositories/:repoId/features                         list features (Bearer)
POST   /v1/repositories/:repoId/features                         create feature (Bearer)
GET    /v1/repositories/:repoId/features/:featureId              get feature (Bearer)
PATCH  /v1/repositories/:repoId/features/:featureId              update feature (Bearer)
DELETE /v1/repositories/:repoId/features/:featureId              delete feature (Bearer)
PATCH  /v1/repositories/:repoId/features/:featureId/position     update position (Bearer)
GET    /v1/repositories/:repoId/features/:featureId/comments     list comments (Bearer)
POST   /v1/repositories/:repoId/features/:featureId/comments     create comment (Bearer)
POST   /v1/repositories/:repoId/features/:featureId/vote         toggle vote (Bearer)
POST   /v1/repositories/:repoId/features/:featureId/dependencies add dependency (Bearer)
DELETE /v1/repositories/:repoId/features/:featureId/dependencies/:dependsOn  remove dep (Bearer)
PATCH  /v1/repositories/:repoId/portal                           update portal visibility (Bearer)

POST   /v1/features/:featureId/runs                    create run (Bearer) — triggers agent pipeline

GET    /v1/portal/:orgSlug/:repoName                                 public portal page (no auth)
GET    /v1/portal/:orgSlug/:repoName/events                          SSE event stream (no auth)
POST   /v1/portal/:orgSlug/:repoName/features/:featureId/vote        toggle vote (no auth)
GET    /v1/portal/:orgSlug/:repoName/features/:featureId/comments    list comments (no auth)
POST   /v1/portal/:orgSlug/:repoName/features/:featureId/comments    create comment (no auth)
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

7 migrations in `server/db/migrations/`:

| Migration | Content                                                                              |
| --------- | ------------------------------------------------------------------------------------ |
| 000001    | Init: users, organizations, org members, repositories, features, comments, votes, deps, refresh tokens, auth codes, github connections |
| 000002    | `portal_public` column on repositories                                               |
| 000003    | `description` column on repositories                                                 |
| 000004    | Split feature status into `review_status` + `build_status` enums                     |
| 000005    | `feature_runs` table + `feature_run_status` enum (pending, running, completed, failed) |
| 000006    | River job queue tables (river_jobs, river_clients, river_queues, river_leaders)       |
| 000007    | `workspace` column on feature_runs                                                   |

Run `make migrate-up` from `server/` (requires `DATABASE_URL` in env).

## Key Implementation Details

- **sqlc**: all store queries are type-safe and generated. Run `make generate` to regenerate after changing `db/queries/*.sql`.
- **River job queue**: PostgreSQL-backed, runs in-process. Client created via `jobs.NewClient()`, workers registered via `workers.Register()`. Jobs dispatched with `jc.Insert()` or `jc.InsertTx()` (transactional).
- **Agent execution**: `ClaudeRunner` spawns `claude -p "<prompt>" --verbose` in a git worktree directory, streaming stdout/stderr with a 1MB buffer. 30-minute timeout per run.
- **Git worktree management**: `RunService.RunAgent()` calls `prepareWorktree()` which is idempotent — removes existing worktree/branch before creating. Branch name: `feature-{featureID}-run-{runID}`. Workspace: `{WORKSPACE_DIR}/{orgID}/{repoID}/worktrees/run-{runID}`.
- **Hub (real-time events)**: in-memory pub/sub keyed by repoID. Portal uses SSE (`/events` endpoint) with `hub.Subscribe()` to push `feature_created`/`feature_updated` events.
- **JWT claims**: `dtos.Claims` embeds `jwt.RegisteredClaims` and carries `UserID` + `Email`. Validated by `middleware.RequireAuth`.
- **PKCE**: server-side verification in `AuthService.ExchangeCode` — computes SHA-256 of verifier and compares to stored challenge.
- **Refresh token storage**: only the SHA-256 hash is stored in `refresh_tokens`; the raw token is sent to the client once and never stored.
- **GitHub token storage**: AES-256-GCM encrypted, then base64-encoded, stored as `TEXT` in `github_connections`. `GithubTokenSource` decrypts on-the-fly, auto-refreshes expired tokens, and re-encrypts + upserts.
- **Request IDs**: `middleware.AddRequestID` sets a UUID per request; `request.GetRequestID` retrieves it for logging.
- **Logging**: `go.uber.org/zap` with colored console output; named loggers per component (e.g., `logger.New().Named("api")`).
- **Module path**: `github.com/didrikolofsson/github-vote-llm`

## Startup Sequence (main.go)

```
Load env → pgxpool → river.Workers → river.Client (jobs.NewClient) → ClaudeRunner →
store.Queries → Hub → Services (wires all deps) → workers.Register() → jc.Start() →
Handlers (wires services) → Gin router (api.New) → HTTP server → graceful shutdown
```

## Server Makefile Targets

```
make dev            # air (live reload)
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
- `github.com/riverqueue/river` — PostgreSQL-based job queue
- `github.com/caarlos0/env/v11` — env var parsing
- `go.uber.org/zap` — structured logging
- `github.com/golang-jwt/jwt/v5` — JWT signing/verification
- `github.com/google/uuid` — request ID generation
- `github.com/google/go-github/v84` — GitHub API client
- `golang.org/x/oauth2` — OAuth2 client (GitHub token exchange + auto-refresh)
- `github.com/joho/godotenv` — `.env` file loading in dev
- `sqlc` — SQL → Go code generation
- `golang-migrate` CLI — migration runner

**Client:**
- React + Vite + TypeScript
- React Router v6
- TanStack Query (react-query)
- React Flow (@xyflow/react) — roadmap canvas
- Tailwind CSS + shadcn/ui
- Zod — runtime schema validation
