package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os/exec"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v84/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

var (
	ErrGitHubNotConnected     = errors.New("github: no connection found for user")
	ErrGitHubTokenUnavailable = errors.New("github: token unavailable or refresh failed")
)

type GithubServiceConfigParams struct {
	TokenEncryptionKey string
	Config             oauth2.Config
}

type GithubService interface {
	Callback(ctx context.Context, code string, userID int64, tokenEncryptionKey string) error
	Status(ctx context.Context, userID int64) (*GithubUserResponse, error)
	Disconnect(ctx context.Context, userID int64) error
	ListReposByAuthenticatedUser(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error)
	CloneRepoToWorkspace(ctx context.Context, userID int64, owner string, name string, workspace string) error
}

type GithubServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
	p  *GithubServiceConfigParams
}

func NewGithubService(db *pgxpool.Pool, q *store.Queries, p *GithubServiceConfigParams) GithubService {
	return &GithubServiceImpl{db: db, q: q, p: p}
}

func (s *GithubServiceImpl) Callback(ctx context.Context, code string, userID int64, tokenEncryptionKey string) error {
	token, err := s.p.Config.Exchange(ctx, code)
	if err != nil {
		return err
	}

	encryptedAccessToken, err := encryption.Encrypt([]byte(token.AccessToken), tokenEncryptionKey)
	if err != nil {
		return err
	}
	var encryptedRefreshToken *string
	if token.RefreshToken != "" {
		encoded, encErr := encryption.Encrypt([]byte(token.RefreshToken), tokenEncryptionKey)
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

func (s *GithubServiceImpl) Status(ctx context.Context, userID int64) (*GithubUserResponse, error) {
	if _, err := s.q.GetGitHubConnectionByUserID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGitHubNotConnected
		}
		return nil, err
	}
	ts := gh.NewGithubTokenSource(
		ctx,
		s.q,
		&s.p.Config,
		userID,
		s.p.TokenEncryptionKey,
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

func (s *GithubServiceImpl) Disconnect(ctx context.Context, userID int64) error {
	conn, err := s.q.GetGitHubConnectionByUserID(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Best-effort: revoke the token on GitHub so the OAuth consent screen
	// appears again on the next connect attempt.
	if err == nil {
		if decrypted, decErr := encryption.Decrypt(conn.AccessTokenEncrypted, s.p.TokenEncryptionKey); decErr == nil {
			_ = s.revokeGitHubToken(ctx, string(decrypted))
		}
	}

	return s.q.DeleteGitHubConnection(ctx, userID)
}

func (s *GithubServiceImpl) revokeGitHubToken(ctx context.Context, accessToken string) error {
	body, err := json.Marshal(map[string]string{"access_token": accessToken})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		"https://api.github.com/applications/"+s.p.Config.ClientID+"/token",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.p.Config.ClientID, s.p.Config.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *GithubServiceImpl) ListReposByAuthenticatedUser(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error) {
	if _, err := s.q.GetGitHubConnectionByUserID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrGitHubNotConnected
		}
		return nil, false, err
	}

	ts := gh.NewGithubTokenSource(
		ctx,
		s.q,
		&s.p.Config,
		userID,
		s.p.TokenEncryptionKey,
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
)

func (s *GithubServiceImpl) CloneRepoToWorkspace(
	ctx context.Context, userID int64, owner, name, workspace string,
) error {
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
		&s.p.Config,
		userID,
		s.p.TokenEncryptionKey,
	)
	tok, err := ts.Token()
	if err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return ErrGitHubTokenUnavailable
	}

	client := gh.NewGithubClientByUserID(ctx, ts)

	repo, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return err
	}
	if repo.CloneURL == nil || *repo.CloneURL == "" {
		return ErrInvalidCloneURL
	}

	u, err := url.Parse(*repo.CloneURL)
	if err != nil {
		return err
	}
	u.User = url.UserPassword("x-access-token", tok.AccessToken)
	authCloneURL := u.String()

	cmd := exec.CommandContext(ctx, "git", "clone", authCloneURL)
	cmd.Dir = workspace
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
