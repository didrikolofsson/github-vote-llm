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

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/githubapp"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

var (
	ErrGitHubNotInstalled    = errors.New("github: no installation found for organization")
	ErrGitHubTokenFailed     = errors.New("github: failed to mint installation token")
	ErrInvalidCloneURL       = errors.New("github: invalid or missing clone URL")
	ErrRunNotFound           = errors.New("github: run not found")
	ErrUserHasNoOrg          = errors.New("github: user has no organization")
	ErrInstallationSuspended = errors.New("github: installation is suspended")
)

type GithubService struct {
	db  *pgxpool.Pool
	q   *store.Queries
	app *githubapp.Client
	env *config.Environment
	jc  *river.Client[pgx.Tx]
}

func NewGithubService(db *pgxpool.Pool, q *store.Queries, app *githubapp.Client, env *config.Environment, jc *river.Client[pgx.Tx]) *GithubService {
	return &GithubService{db: db, q: q, app: app, env: env, jc: jc}
}

// CreateInstallURL returns the github.com URL where the user will install the app,
// with a signed single-use state token bound to the user's session.
func (s *GithubService) CreateInstallURL(ctx context.Context, userID int64) (string, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	org, err := qtx.GetOrganizationMembershipByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	state, err := githubapp.CreateInstallStateToken(
		ctx, org.OrganizationID, userID, s.env.JWT_SECRET,
	)
	if err != nil {
		return "", err
	}
	_, err = qtx.UpsertInstallation(ctx, store.UpsertInstallationParams{
		OrganizationID:    org.OrganizationID,
		InstalledByUserID: &userID,
		State:             store.GithubInstallationStatePending,
	})
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	v := url.Values{}
	v.Set("state", state)
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?%s", s.env.GITHUB_APP_SLUG, v.Encode()), nil
}

type InstallationStatus struct {
	Installed           bool
	Login               string
	AccountType         string
	RepositorySelection string
	Suspended           bool
	InstallationID      int64
}

// GetInstallationStatus returns the current install status for the user's organization.
func (s *GithubService) GetInstallationStatus(ctx context.Context, userID int64) (*InstallationStatus, error) {
	orgID, err := s.orgIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	inst, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &InstallationStatus{Installed: false}, nil
		}
		return nil, err
	}
	return &InstallationStatus{
		Installed:           true,
		Login:               inst.GithubAccountLogin,
		AccountType:         inst.GithubAccountType,
		RepositorySelection: inst.RepositorySelection,
		Suspended:           inst.SuspendedAt.Valid,
		InstallationID:      inst.GithubInstallationID,
	}, nil
}

// CompleteInstall validates the state token, fetches installation details from GitHub,
// and persists the installation against the user's organization.
func (s *GithubService) CompleteInstall(ctx context.Context, installationID int64, state githubapp.InstallStateClaims) error {
	userID, err := githubapp.ConsumeInstallState(ctx, s.q, state)
	if err != nil {
		return err
	}

	orgID, err := s.orgIDForUser(ctx, userID)
	if err != nil {
		return err
	}

	inst, err := s.app.GetInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("fetch installation: %w", err)
	}
	if inst == nil || inst.Account == nil {
		return fmt.Errorf("github: installation %d returned empty metadata", installationID)
	}

	var suspendedAt pgtype.Timestamptz
	if inst.SuspendedAt != nil {
		suspendedAt = pgtype.Timestamptz{Time: inst.SuspendedAt.Time, Valid: true}
	}

	saved, err := s.q.UpsertInstallation(ctx, store.UpsertInstallationParams{
		OrganizationID:       orgID,
		GithubInstallationID: installationID,
		GithubAccountLogin:   inst.Account.GetLogin(),
		GithubAccountID:      inst.Account.GetID(),
		GithubAccountType:    inst.Account.GetType(),
		RepositorySelection:  inst.GetRepositorySelection(),
		SuspendedAt:          suspendedAt,
		InstalledByUserID:    &userID,
	})
	if err != nil {
		return err
	}

	if err := s.syncInstallationRepositories(ctx, saved); err != nil {
		return fmt.Errorf("sync repos: %w", err)
	}
	return nil
}

// DeleteInstallation removes the installation row for the user's organization.
// Does not uninstall on GitHub — if the user wants to revoke, they must do so on github.com;
// the webhook will then reconcile. This call is DB-only.
func (s *GithubService) DeleteInstallation(ctx context.Context, userID int64) error {
	orgID, err := s.orgIDForUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.q.DeleteInstallationByOrgID(ctx, orgID)
}

// ListInstallationRepositories lists repos accessible to the installation.
func (s *GithubService) ListInstallationRepositories(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error) {
	orgID, err := s.orgIDForUser(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	inst, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrGitHubNotInstalled
		}
		return nil, false, err
	}
	if inst.SuspendedAt.Valid {
		return nil, false, ErrInstallationSuspended
	}

	client := s.app.InstallationGithubClient(ctx, inst.GithubInstallationID)
	repos, resp, err := client.Apps.ListRepos(ctx, &github.ListOptions{Page: page, PerPage: 30})
	if err != nil {
		return nil, false, err
	}
	out := make([]dtos.GitHubRepository, 0, len(repos.Repositories))
	for _, r := range repos.Repositories {
		out = append(out, dtos.GitHubRepository{
			Owner: r.Owner.GetLogin(),
			Repo:  r.GetName(),
		})
	}
	return out, resp.NextPage > 0, nil
}

// CloneRepoToWorkspace clones the repo for a run, authenticating as the installation
// attached to the run's organization.
func (s *GithubService) CloneRepoToWorkspace(ctx context.Context, runID int64) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}

	repoPath := filepath.Join(run.Workspace, run.RepositoryName)
	if _, err := os.Stat(repoPath); err == nil {
		if _, err := s.jc.Insert(ctx, args.RunAgentArgs{RunID: runID}, nil); err != nil {
			return err
		}
		return nil
	}

	token, err := s.installationTokenForOrg(ctx, run.OrganizationID)
	if err != nil {
		return err
	}

	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git",
		token, run.RepositoryOwner, run.RepositoryName)

	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL)
	cmd.Dir = run.Workspace
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %w: %s", err, redactToken(string(out), token))
	}

	if _, err := s.jc.Insert(ctx, args.RunAgentArgs{RunID: runID}, nil); err != nil {
		return err
	}
	return nil
}

// PushBranch pushes the given branch from worktreeDir to GitHub using a fresh installation token.
func (s *GithubService) PushBranch(ctx context.Context, orgID int64, worktreeDir, owner, name, branch string) error {
	token, err := s.installationTokenForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	pushURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, name)
	push := exec.CommandContext(ctx, "git", "-C", worktreeDir, "push", pushURL, branch)
	if combined, err := push.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, redactToken(string(combined), token))
	}
	return nil
}

// OpenPR opens a pull request as the GitHub App installation.
func (s *GithubService) OpenPR(ctx context.Context, orgID int64, owner, name, branch, title, body string) (string, error) {
	inst, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		return "", err
	}
	client := s.app.InstallationGithubClient(ctx, inst.GithubInstallationID)

	repo, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return "", err
	}
	defaultBranch := repo.GetDefaultBranch()
	pr, _, err := client.PullRequests.Create(ctx, owner, name, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &branch,
		Base:  &defaultBranch,
	})
	if err != nil {
		return "", err
	}
	return pr.GetHTMLURL(), nil
}

// installationTokenForOrg resolves orgID -> installation -> fresh installation access token.
func (s *GithubService) installationTokenForOrg(ctx context.Context, orgID int64) (string, error) {
	inst, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrGitHubNotInstalled
		}
		return "", err
	}
	if inst.SuspendedAt.Valid {
		return "", ErrInstallationSuspended
	}
	tok, _, err := s.app.InstallationToken(ctx, inst.GithubInstallationID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGitHubTokenFailed, err)
	}
	return tok, nil
}

func (s *GithubService) orgIDForUser(ctx context.Context, userID int64) (int64, error) {
	m, err := s.q.GetOrganizationMembershipByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUserHasNoOrg
		}
		return 0, err
	}
	return m.OrganizationID, nil
}

// syncInstallationRepositories refreshes the cached repo list for a given installation row.
// For repository_selection="all" we still cache the visible list on install.
func (s *GithubService) syncInstallationRepositories(ctx context.Context, inst store.GithubInstallation) error {
	client := s.app.InstallationGithubClient(ctx, inst.GithubInstallationID)

	if err := s.q.ClearInstallationRepositories(ctx, inst.ID); err != nil {
		return err
	}

	page := 1
	for {
		resp, httpResp, err := client.Apps.ListRepos(ctx, &github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return err
		}
		for _, r := range resp.Repositories {
			if err := s.q.AddInstallationRepository(ctx, store.AddInstallationRepositoryParams{
				InstallationID:     inst.ID,
				GithubRepositoryID: r.GetID(),
				RepositoryName:     r.GetName(),
				RepositoryFullName: r.GetFullName(),
			}); err != nil {
				return err
			}
		}
		if httpResp.NextPage == 0 {
			break
		}
		page = httpResp.NextPage
	}
	return nil
}

// redactToken scrubs a token value from a string, useful before logging git output.
func redactToken(s, token string) string {
	if token == "" {
		return s
	}
	// Avoid allocating a replacer for short strings.
	return replaceAll(s, token, "***")
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			out = append(out, new...)
			i += len(old)
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}

// Webhook handlers — called by the webhook handler.

// HandleInstallationEvent updates DB state based on `installation` webhook events.
func (s *GithubService) HandleInstallationEvent(ctx context.Context, event *github.InstallationEvent) error {
	if event.Installation == nil {
		return fmt.Errorf("installation event missing installation")
	}
	id := event.Installation.GetID()
	action := event.GetAction()

	switch action {
	case "created":
		// Idempotent with the callback: if the row exists (callback beat us), do nothing.
		if _, err := s.q.GetInstallationByGithubID(ctx, id); err == nil {
			return nil
		}
		// Otherwise: webhook arrived first (rare); we still need orgID to attach it.
		// Without a user session we can't determine the org. Skip — the callback will upsert.
		return nil

	case "deleted":
		return s.q.DeleteInstallationByGithubID(ctx, id)

	case "suspend":
		ts := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if event.Installation.SuspendedAt != nil {
			ts.Time = event.Installation.SuspendedAt.Time
		}
		return s.q.SetInstallationSuspendedByGithubID(ctx, store.SetInstallationSuspendedByGithubIDParams{
			GithubInstallationID: id,
			SuspendedAt:          ts,
		})

	case "unsuspend":
		return s.q.SetInstallationSuspendedByGithubID(ctx, store.SetInstallationSuspendedByGithubIDParams{
			GithubInstallationID: id,
			SuspendedAt:          pgtype.Timestamptz{Valid: false},
		})
	}
	return nil
}

// HandleInstallationRepositoriesEvent keeps the repo cache in sync.
func (s *GithubService) HandleInstallationRepositoriesEvent(ctx context.Context, event *github.InstallationRepositoriesEvent) error {
	if event.Installation == nil {
		return fmt.Errorf("installation_repositories event missing installation")
	}
	inst, err := s.q.GetInstallationByGithubID(ctx, event.Installation.GetID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	for _, r := range event.RepositoriesAdded {
		if err := s.q.AddInstallationRepository(ctx, store.AddInstallationRepositoryParams{
			InstallationID:     inst.ID,
			GithubRepositoryID: r.GetID(),
			RepositoryName:     r.GetName(),
			RepositoryFullName: r.GetFullName(),
		}); err != nil {
			return err
		}
	}
	for _, r := range event.RepositoriesRemoved {
		if err := s.q.RemoveInstallationRepository(ctx, store.RemoveInstallationRepositoryParams{
			InstallationID:     inst.ID,
			GithubRepositoryID: r.GetID(),
		}); err != nil {
			return err
		}
	}
	return nil
}
