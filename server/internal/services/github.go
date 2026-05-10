package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	appgithub "github.com/didrikolofsson/github-vote-llm/internal/github"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
	gh "github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInstallationNotFound  = errors.New("github app installation not found")
	ErrInstallationSuspended = errors.New("github app installation is suspended")
)

type appInstallStateClaims struct {
	OrgID  int64 `json:"org_id"`
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type GithubService struct {
	db        *pgxpool.Pool
	q         *store.Queries
	env       *config.Environment
	appClient *appgithub.AppClient
	hub       hub.Hub
}

type GithubServiceDeps struct {
	DB        *pgxpool.Pool
	Queries   *store.Queries
	Env       *config.Environment
	AppClient *appgithub.AppClient
	Hub       hub.Hub
}

func NewGithubService(deps GithubServiceDeps) *GithubService {
	return &GithubService{
		db:        deps.DB,
		q:         deps.Queries,
		env:       deps.Env,
		appClient: deps.AppClient,
		hub:       deps.Hub,
	}
}

func (s *GithubService) FrontendURL() string {
	return s.env.FRONTEND_URL
}

// CreateAppInstallURL generates a GitHub App installation URL with a signed state token
// that encodes the org ID, so we can link the installation to the org on callback.
func (s *GithubService) CreateAppInstallURL(ctx context.Context, orgID int64, userID int64) (string, error) {
	claims := appInstallStateClaims{
		OrgID:  orgID,
		UserID: userID,
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
	userID := claims.UserID

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
		InstalledByUserID:    &userID,
	})
	if err != nil {
		return 0, fmt.Errorf("upsert installation: %w", err)
	}

	if !suspendedAt.Valid {
		s.hub.PublishOrg(orgID, hub.EventInstallationActive)
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

	if installation.SuspendedAt.Valid {
		return 0, ErrInstallationSuspended
	}

	s.hub.PublishOrg(installation.OrganizationID, hub.EventInstallationActive)

	return installation.OrganizationID, nil
}

type GithubAccountType string

func (s *GithubService) GetInstallationStatus(ctx context.Context, orgID int64) (dtos.AppInstallationStatus, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dtos.AppInstallationStatus{
				Installed: false,
			}, nil
		}
		return dtos.AppInstallationStatus{}, err
	}

	var suspendedAt *time.Time
	if installation.SuspendedAt.Valid {
		t := installation.SuspendedAt.Time
		suspendedAt = &t
	}

	installedByUserName := ""
	if installation.InstalledByUserName != nil {
		installedByUserName = *installation.InstalledByUserName
	}

	return dtos.AppInstallationStatus{
		Installed:           true,
		SuspendedAt:         suspendedAt,
		TargetLogin:         installation.GithubAccountLogin,
		AccountType:         dtos.GithubAccountType(installation.GithubAccountType),
		InstalledByUserName: installedByUserName,
	}, nil
}

func (s *GithubService) GetInstallationByOrgID(ctx context.Context, orgID int64) (dtos.AppInstallation, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dtos.AppInstallation{}, nil
		}
		return dtos.AppInstallation{}, err
	}
	return dtos.AppInstallation{
		ID:                   installation.ID,
		OrganizationID:       installation.OrganizationID,
		GithubInstallationID: installation.GithubInstallationID,
		GithubAccountLogin:   installation.GithubAccountLogin,
		GithubAccountID:      installation.GithubAccountID,
		GithubAccountType:    installation.GithubAccountType,
		RepositorySelection:  installation.RepositorySelection,
		SuspendedAt:          &installation.SuspendedAt.Time,
		InstalledByUserID:    installation.InstalledByUserID,
		CreatedAt:            installation.CreatedAt.Time,
		UpdatedAt:            installation.UpdatedAt.Time,
	}, nil
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
	installationID := payload.Installation.ID

	existing, err := s.q.GetInstallationByInstallationID(ctx, installationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get installation by id: %w", err)
	}
	orgID := existing.OrganizationID

	switch payload.Action {
	case "deleted":
		if err := s.q.DeleteInstallationByInstallationID(ctx, installationID); err != nil {
			return err
		}
		s.hub.PublishOrg(orgID, hub.EventInstallationRemoved)
		return nil

	case "suspend":
		t := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if err := s.q.SetInstallationSuspendedByInstallationID(ctx, store.SetInstallationSuspendedByInstallationIDParams{
			GithubInstallationID: installationID,
			SuspendedAt:          t,
		}); err != nil {
			return err
		}
		s.hub.PublishOrg(orgID, hub.EventInstallationSuspended)
		return nil

	case "unsuspend":
		if err := s.q.SetInstallationSuspendedByInstallationID(ctx, store.SetInstallationSuspendedByInstallationIDParams{
			GithubInstallationID: installationID,
			SuspendedAt:          pgtype.Timestamptz{Valid: false},
		}); err != nil {
			return err
		}
		s.hub.PublishOrg(orgID, hub.EventInstallationActive)
		return nil
	}
	// "created" and "new_permissions_accepted" are handled by the redirect callback flow.
	return nil
}

// CloneRepoToWorkspace clones a GitHub repository using an installation token.
// It is idempotent — if the repo directory already exists it is skipped.
func (s *GithubService) CloneRepoToWorkspace(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	installation, err := s.q.GetInstallationByOrgID(ctx, run.OrganizationID)
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

	//nolint:gosec // git clone URL embeds installation-scoped token; destination is workspace-bound repoDir.
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %w: %s", err, out)
	}
	return nil
}

// PushBranch pushes a local branch to GitHub using an installation token.
func (s *GithubService) PushBranch(ctx context.Context, orgID int64, worktreeDir, owner, repo, branch string) error {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get installation: %w", err)
	}

	token, err := s.appClient.InstallationToken(ctx, installation.GithubInstallationID)
	if err != nil {
		return fmt.Errorf("get installation token: %w", err)
	}

	remote := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)
	//nolint:gosec // git arguments come from server-controlled workspace paths and GitHub metadata.
	cmd := exec.CommandContext(ctx, "git", "-C", worktreeDir, "push", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, out)
	}
	return nil
}

// OpenPR creates a pull request on GitHub using the App installation client.
// It ensures the "ai-generated" label exists on the repo and applies it to the PR.
func (s *GithubService) OpenPR(ctx context.Context, orgID int64, owner, repo, branch, title, body string) (string, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
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

func (s *GithubService) ListInstallationRepositories(ctx context.Context, orgID int64, page int64) ([]dtos.GitHubRepository, bool, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrInstallationNotFound
		}
		return nil, false, fmt.Errorf("get installation by id: %w", err)
	}

	if installation.SuspendedAt.Valid {
		return nil, false, ErrInstallationSuspended
	}

	client, err := s.appClient.InstallationClient(ctx, installation.GithubInstallationID)
	if err != nil {
		// Need a logger on this
		return nil, false, fmt.Errorf("create installation client: %w", err)
	}

	if page == 0 {
		page = 1
	}

	// Installation tokens must use /installation/repositories, not /user/repos.
	list, resp, err := client.Apps.ListRepos(ctx, &gh.ListOptions{
		PerPage: 100,
		Page:    int(page),
	})
	if err != nil {
		return nil, false, fmt.Errorf("list installation repositories: %w", err)
	}

	hasMore := resp != nil && resp.NextPage != 0
	var repos []*gh.Repository
	if list != nil {
		repos = list.Repositories
	}

	return githubReposToDTOs(repos), hasMore, nil
}

func githubReposToDTOs(repos []*gh.Repository) []dtos.GitHubRepository {
	out := make([]dtos.GitHubRepository, len(repos))
	for i, r := range repos {
		out[i] = dtos.GitHubRepository{
			GithubRepositoryID: r.GetID(),
			Owner:              r.GetOwner().GetLogin(),
			Name:               r.GetName(),
			FullName:           r.GetFullName(),
			Private:            r.GetPrivate(),
		}
	}
	return out
}
