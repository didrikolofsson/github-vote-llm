package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	appgithub "github.com/didrikolofsson/github-vote-llm/internal/github"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gitauth_account "github.com/didrikolofsson/github-vote-llm/internal/gitauth/account"
	gitauth_client "github.com/didrikolofsson/github-vote-llm/internal/gitauth/client"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
	gh "github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

var (
	ErrInstallationNotFound  = errors.New("github app installation not found")
	ErrInstallationNotActive = errors.New("github app installation is not active")
)

type appInstallStateClaims struct {
	OrgID int64 `json:"org_id"`
	jwt.RegisteredClaims
}

type GithubService struct {
	db            *pgxpool.Pool
	q             *store.Queries
	accountClient *gitauth_account.GithubAccountClient
	env           *config.Environment
	cfg           *oauth2.Config
	appClient     *appgithub.AppClient
}

type GithubServiceDeps struct {
	DB            *pgxpool.Pool
	Queries       *store.Queries
	Env           *config.Environment
	AccountClient *gitauth_account.GithubAccountClient
	Config        *oauth2.Config
	AppClient     *appgithub.AppClient
}

func NewGithubService(deps GithubServiceDeps) *GithubService {
	return &GithubService{
		db:            deps.DB,
		q:             deps.Queries,
		accountClient: deps.AccountClient,
		env:           deps.Env,
		cfg:           deps.Config,
		appClient:     deps.AppClient,
	}
}

func (s *GithubService) FrontendURL() string {
	return s.env.FRONTEND_URL
}

func (s *GithubService) CreateAuthURL(ctx context.Context, userID int64) (string, error) {
	authUrl, err := s.accountClient.CreateAuthURL(ctx, userID)
	if err != nil {
		return "", err
	}
	return authUrl, nil
}

func (s *GithubService) ExchangeCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	config := gitauth_client.NewOauthConfig(gitauth_client.OauthConfigParams{
		ClientID:     s.env.GITHUB_CLIENT_ID,
		ClientSecret: s.env.GITHUB_CLIENT_SECRET,
		Scopes:       []string{"user:email", "read:org"},
		RedirectURL:  fmt.Sprintf("%s/github/auth/callback", s.env.SERVER_URL),
	})

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *GithubService) VerifyAuthStateToken(ctx context.Context, token string) (gitauth_account.AuthStateClaims, error) {
	claims, err := s.accountClient.VerifyAuthStateToken(ctx, token)
	if err != nil {
		return gitauth_account.AuthStateClaims{}, err
	}
	return claims, nil
}

func (s *GithubService) UpsertGithubAccountTokenByUserID(ctx context.Context, userID int64, token *oauth2.Token) error {
	return s.accountClient.UpsertGithubAccountTokenByUserID(ctx, userID, token)
}

func (s *GithubService) GetAccountByUserID(ctx context.Context, userID int64) (string, error) {
	ts := gitauth_client.NewGithubTokenSource(gitauth_client.GithubTokenSourceDeps{
		DB:      s.db,
		Queries: s.q,
		UserID:  userID,
		Config:  s.cfg,
	})
	client := gitauth_client.NewGithubClient(ctx, ts)
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}

// CreateAppInstallURL generates a GitHub App installation URL with a signed state token
// that encodes the org ID, so we can link the installation to the org on callback.
func (s *GithubService) CreateAppInstallURL(ctx context.Context, orgID int64) (string, error) {
	claims := appInstallStateClaims{
		OrgID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(s.env.JWT_SECRET))
	if err != nil {
		return "", fmt.Errorf("sign state token: %w", err)
	}
	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s",
		s.env.GITHUB_APP_SLUG, url.QueryEscape(state))
	return installURL, nil
}

// HandleAppInstallCallback validates the state JWT, fetches installation details from GitHub,
// and upserts the installation record linked to the org.
func (s *GithubService) HandleAppInstallCallback(ctx context.Context, installationID int64, state string) (int64, error) {
	// Validate state and extract org ID.
	claims := &appInstallStateClaims{}
	_, err := jwt.ParseWithClaims(state, claims, func(t *jwt.Token) (any, error) {
		return []byte(s.env.JWT_SECRET), nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid state token: %w", err)
	}

	orgID := claims.OrgID

	// Fetch installation details from GitHub to get account info.
	appClient, err := s.appClient.AppAPIClient()
	if err != nil {
		return 0, fmt.Errorf("create app client: %w", err)
	}
	installation, _, err := appClient.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return 0, fmt.Errorf("fetch installation from github: %w", err)
	}

	var suspendedAt pgtype.Timestamptz
	if installation.SuspendedAt != nil {
		suspendedAt = pgtype.Timestamptz{Time: installation.SuspendedAt.Time, Valid: true}
	}

	repoSelection := "all"
	if installation.RepositorySelection != nil {
		repoSelection = *installation.RepositorySelection
	}

	_, err = s.q.UpsertInstallation(ctx, store.UpsertInstallationParams{
		OrganizationID:       orgID,
		GithubInstallationID: installationID,
		GithubAccountLogin:   installation.GetAccount().GetLogin(),
		GithubAccountID:      installation.GetAccount().GetID(),
		GithubAccountType:    installation.GetAccount().GetType(),
		RepositorySelection:  repoSelection,
		SuspendedAt:          suspendedAt,
		State:                store.GithubInstallationStateActive,
	})
	if err != nil {
		return 0, fmt.Errorf("upsert installation: %w", err)
	}

	return orgID, nil
}

func (s *GithubService) HandleAppUpdateCallback(ctx context.Context, installationID int64) (int64, error) {
	installation, err := s.q.GetInstallationByInstallationID(ctx, installationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInstallationNotFound
		}
		return 0, fmt.Errorf("get installation by id: %w", err)
	}

	if installation.State != store.GithubInstallationStateActive {
		return 0, ErrInstallationNotActive
	}
	return installation.OrganizationID, nil
}

type GithubAccountType string

const (
	GithubAccountTypeUser         GithubAccountType = "user"
	GithubAccountTypeOrganization GithubAccountType = "organization"
)

type AppInstallationStatus struct {
	Installed   bool
	TargetLogin string
	SuspendedAt *time.Time
	AccountType GithubAccountType
}

// GetInstallationStatus returns the installation status for an org, doing a live GitHub API
// verification. If GitHub returns 404 the stale record is deleted (self-healing).
func (s *GithubService) GetInstallationStatus(ctx context.Context, orgID int64) (AppInstallationStatus, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AppInstallationStatus{Installed: false}, nil
		}
		return AppInstallationStatus{}, err
	}

	// Live-verify against GitHub.
	appClient, err := s.appClient.AppAPIClient()
	if err != nil {
		return AppInstallationStatus{}, fmt.Errorf("create app client: %w", err)
	}
	_, resp, err := appClient.Apps.GetInstallation(ctx, installation.GithubInstallationID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Installation was removed on GitHub — delete the stale record.
			_ = s.q.DeleteInstallationByOrgID(ctx, orgID)
			return AppInstallationStatus{Installed: false}, nil
		}
		return AppInstallationStatus{}, fmt.Errorf("verify installation with github: %w", err)
	}

	accountType := GithubAccountTypeOrganization
	if installation.GithubAccountType == "User" {
		accountType = GithubAccountTypeUser
	}
	status := AppInstallationStatus{
		Installed:   true,
		TargetLogin: installation.GithubAccountLogin,
		AccountType: accountType,
	}
	if installation.SuspendedAt.Valid {
		status.SuspendedAt = &installation.SuspendedAt.Time
	}
	return status, nil
}

// InstallationWebhookPayload is a minimal representation of GitHub's installation event.
type InstallationWebhookPayload struct {
	Action       string `json:"action"`
	Installation struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			ID    int64  `json:"id"`
			Type  string `json:"type"`
		} `json:"account"`
		RepositorySelection string  `json:"repository_selection"`
		SuspendedAt         *string `json:"suspended_at"`
	} `json:"installation"`
}

// HandleInstallationWebhook syncs installation state from GitHub webhook events.
func (s *GithubService) HandleInstallationWebhook(ctx context.Context, payload InstallationWebhookPayload) error {
	githubID := payload.Installation.ID
	switch payload.Action {
	case "deleted":
		return s.q.DeleteInstallationByGithubID(ctx, githubID)

	case "suspend":
		t := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		return s.q.SetInstallationSuspendedByGithubID(ctx, store.SetInstallationSuspendedByGithubIDParams{
			GithubInstallationID: githubID,
			SuspendedAt:          t,
		})

	case "unsuspend":
		return s.q.SetInstallationSuspendedByGithubID(ctx, store.SetInstallationSuspendedByGithubIDParams{
			GithubInstallationID: githubID,
			SuspendedAt:          pgtype.Timestamptz{Valid: false},
		})
	}
	// "created" and "new_permissions_accepted" are handled by the redirect callback flow.
	return nil
}

// GetInstallationByOrgID returns the stored installation record for an org.
func (s *GithubService) GetInstallationByOrgID(ctx context.Context, orgID int64) (store.GithubInstallation, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.GithubInstallation{}, ErrInstallationNotFound
		}
		return store.GithubInstallation{}, err
	}
	return installation, nil
}

// CloneRepoToWorkspace clones a GitHub repository using an installation token.
// It is idempotent — if the repo directory already exists it is skipped.
func (s *GithubService) CloneRepoToWorkspace(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	installation, err := s.GetInstallationByOrgID(ctx, run.OrganizationID)
	if err != nil {
		return fmt.Errorf("get installation for org %d: %w", run.OrganizationID, err)
	}

	token, err := s.appClient.InstallationToken(ctx, installation.GithubInstallationID)
	if err != nil {
		return fmt.Errorf("get installation token: %w", err)
	}

	repoDir := filepath.Join(run.Workspace, run.RepositoryName)

	// Check if already cloned.
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		return nil
	}

	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git",
		token, run.RepositoryOwner, run.RepositoryName)

	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %w: %s", err, out)
	}
	return nil
}

// PushBranch pushes a local branch to GitHub using an installation token.
func (s *GithubService) PushBranch(ctx context.Context, orgID int64, worktreeDir, owner, repo, branch string) error {
	installation, err := s.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get installation: %w", err)
	}

	token, err := s.appClient.InstallationToken(ctx, installation.GithubInstallationID)
	if err != nil {
		return fmt.Errorf("get installation token: %w", err)
	}

	remote := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)
	cmd := exec.CommandContext(ctx, "git", "-C", worktreeDir, "push", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, out)
	}
	return nil
}

// OpenPR creates a pull request on GitHub using the App installation client.
// It ensures the "ai-generated" label exists on the repo and applies it to the PR.
func (s *GithubService) OpenPR(ctx context.Context, orgID int64, owner, repo, branch, title, body string) (string, error) {
	installation, err := s.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("get installation: %w", err)
	}

	client, err := s.appClient.InstallationClient(ctx, installation.GithubInstallationID)
	if err != nil {
		return "", fmt.Errorf("create installation client: %w", err)
	}

	const labelName = "ai-generated"
	const labelColor = "0075ca"
	if err := ensureLabel(ctx, client, owner, repo, labelName, labelColor); err != nil {
		return "", fmt.Errorf("ensure label: %w", err)
	}

	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &gh.NewPullRequest{
		Title:               gh.Ptr(title),
		Head:                gh.Ptr(branch),
		Base:                gh.Ptr("main"),
		Body:                gh.Ptr(body),
		MaintainerCanModify: gh.Ptr(true),
	})
	if err != nil {
		return "", fmt.Errorf("create pull request: %w", err)
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, owner, repo, pr.GetNumber(), []string{labelName})
	if err != nil {
		return "", fmt.Errorf("add label to pr: %w", err)
	}

	return pr.GetHTMLURL(), nil
}

// ensureLabel creates a GitHub label if it does not already exist on the repo.
func ensureLabel(ctx context.Context, client *gh.Client, owner, repo, name, color string) error {
	_, _, err := client.Issues.GetLabel(ctx, owner, repo, name)
	if err == nil {
		return nil
	}
	_, _, createErr := client.Issues.CreateLabel(ctx, owner, repo, &gh.Label{
		Name:  gh.Ptr(name),
		Color: gh.Ptr(color),
	})
	return createErr
}
