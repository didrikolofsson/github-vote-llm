package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	gha "golang.org/x/oauth2/github"
)

var (
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

type GithubService struct {
	db  *pgxpool.Pool
	q   *store.Queries
	cfg *oauth2.Config
	env *config.Environment
}

func NewGithubService(db *pgxpool.Pool, q *store.Queries, env *config.Environment) *GithubService {
	cfg := gh.NewGithubOAuthConfig(
		env.GITHUB_CLIENT_ID,
		env.GITHUB_CLIENT_SECRET,
		env.SERVER_URL+"/v1/github/callback",
	)
	return &GithubService{db: db, q: q, cfg: cfg, env: env}
}

// oauthStateClaims is signed into the GitHub `state` query param so /callback can bind the code to a user.
type oauthStateClaims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

var (
	ErrFailedToBuildOAuthState    = errors.New("github: failed to build oauth state")
	ErrInvalidOrExpiredOAuthState = errors.New("github: invalid or expired oauth state")
)

// Authorize lets the client initiate the OAuth2 flow by returning the GitHub authorization URL.
// Requires JWT (see api router). Response matches client: { "authorize_url": "..." }.
func (s *GithubService) CreateOAuthState(ctx context.Context, userID int64) (string, error) {
	stateTok := jwt.NewWithClaims(jwt.SigningMethodHS256, oauthStateClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		},
	})
	stateStr, err := stateTok.SignedString([]byte(s.env.JWT_SECRET))
	if err != nil {
		return "", ErrFailedToBuildOAuthState
	}

	v := url.Values{}
	v.Set("client_id", s.env.GITHUB_CLIENT_ID)
	v.Set("redirect_uri", s.env.SERVER_URL+"/v1/github/callback")
	v.Set("scope", "repo read:org")
	v.Set("state", stateStr)
	v.Set("prompt", "select_account")

	authorizeURL := gha.Endpoint.AuthURL + "?" + v.Encode()
	return authorizeURL, nil
}

func (s *GithubService) ReadOAuthStateClaims(ctx context.Context, state string) (*oauthStateClaims, error) {
	var claims oauthStateClaims
	tok, err := jwt.ParseWithClaims(state, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.env.JWT_SECRET), nil
	})
	if err != nil || tok == nil || !tok.Valid {
		return nil, ErrInvalidOrExpiredOAuthState
	}
	return &claims, nil
}

func (s *GithubService) ExchangeCodeForAccessToken(ctx context.Context, code string, userID int64) error {
	token, err := s.cfg.Exchange(ctx, code)
	if err != nil {
		return err
	}

	encryptedAccessToken, err := encryption.Encrypt([]byte(token.AccessToken), s.env.TOKEN_ENCRYPTION_KEY)
	if err != nil {
		return err
	}
	var encryptedRefreshToken *string
	if token.RefreshToken != "" {
		encoded, encErr := encryption.Encrypt([]byte(token.RefreshToken), s.env.TOKEN_ENCRYPTION_KEY)
		if encErr != nil {
			return encErr
		}
		encryptedRefreshToken = &encoded
	}
	var expiresAt pgtype.Timestamptz
	if !token.Expiry.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: token.Expiry, Valid: true}
	}
	_, err = s.q.UpsertGitHubConnection(ctx, store.UpsertGitHubConnectionParams{
		UserID:               userID,
		AccessTokenEncrypted: encryptedAccessToken,
		RefreshToken:         encryptedRefreshToken,
		TokenExpiresAt:       expiresAt,
	})
	return err
}

type GithubUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func (s *GithubService) GetGitHubConnectionStatus(ctx context.Context, userID int64) (*GithubUserResponse, error) {
	if _, err := s.q.GetGitHubConnectionByUserID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGitHubNotConnected
		}
		return nil, err
	}
	ts := gh.NewGithubTokenSource(
		ctx,
		s.q,
		s.cfg,
		userID,
		s.env.TOKEN_ENCRYPTION_KEY,
	)
	client := gh.NewGithubClientByUserID(ctx, ts)
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return &GithubUserResponse{
		ID:    *user.ID,
		Login: *user.Login,
	}, nil
}

func (s *GithubService) DeleteGitHubConnection(ctx context.Context, userID int64) error {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Best-effort: revoke the token on GitHub so the OAuth consent screen
	// appears again on the next connect attempt.
	if err == nil {
		if decrypted, decErr := encryption.Decrypt(conn.AccessTokenEncrypted, s.env.TOKEN_ENCRYPTION_KEY); decErr == nil {
			_ = s.revokeGitHubToken(ctx, string(decrypted))
		}
	}

	return s.q.DeleteGitHubConnection(ctx, userID)
}

func (s *GithubService) revokeGitHubToken(ctx context.Context, accessToken string) error {
	body, err := json.Marshal(map[string]string{"access_token": accessToken})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		"https://api.github.com/applications/"+s.cfg.ClientID+"/token",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *GithubService) ListReposByAuthenticatedUser(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error) {
	if _, err := s.q.GetGitHubConnectionByUserID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrGitHubNotConnected
		}
		return nil, false, err
	}

	ts := gh.NewGithubTokenSource(
		ctx,
		s.q,
		s.cfg,
		userID,
		s.env.TOKEN_ENCRYPTION_KEY,
	)
	client := gh.NewGithubClientByUserID(ctx, ts)

	repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{Page: page, PerPage: 30},
	})
	if err != nil {
		return nil, false, err
	}

	out := make([]dtos.GitHubRepository, len(repos))
	for i, r := range repos {
		out[i] = dtos.GitHubRepository{
			Owner: r.Owner.GetLogin(),
			Repo:  r.GetName(),
		}
	}
	return out, resp.NextPage > 0, nil
}

var (
	ErrInvalidCloneURL = errors.New("github: invalid or missing clone URL")
	ErrRunNotFound     = errors.New("github: run not found")
)

func (s *GithubService) CloneRepoToWorkspace(
	ctx context.Context, userID, runID int64,
) error {
	run, err := s.q.GetRunByID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}

	repo, err := s.q.GetRepositoryByFeatureID(ctx, run.FeatureID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepositoryNotFound
		}
		return err
	}

	// Workspace is created before job is enqueued.
	repoPath := filepath.Join(run.Workspace, repo.Name)
	if _, err := os.Stat(repoPath); err == nil {
		return nil
	}

	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if conn.AccessTokenEncrypted == "" {
		return ErrGitHubNotConnected
	}

	ts := gh.NewGithubTokenSource(
		ctx,
		s.q,
		s.cfg,
		userID,
		s.env.TOKEN_ENCRYPTION_KEY,
	)
	tok, err := ts.Token()
	if err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return ErrGitHubTokenUnavailable
	}

	client := gh.NewGithubClientByUserID(ctx, ts)

	ghRepo, _, err := client.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		return err
	}
	if ghRepo.CloneURL == nil || *ghRepo.CloneURL == "" {
		return ErrInvalidCloneURL
	}

	u, err := url.Parse(*ghRepo.CloneURL)
	if err != nil {
		return err
	}
	u.User = url.UserPassword("x-access-token", tok.AccessToken)
	authCloneURL := u.String()

	cmd := exec.CommandContext(ctx, "git", "clone", authCloneURL)
	cmd.Dir = run.Workspace
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (s *GithubService) OpenPR(
	ctx context.Context, userID int64, owner, name, branch, title, body string,
) (string, error) {
	ts := gh.NewGithubTokenSource(ctx, s.q, s.cfg, userID, s.env.TOKEN_ENCRYPTION_KEY)
	client := gh.NewGithubClientByUserID(ctx, ts)

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
