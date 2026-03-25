package services

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/encryption"
	"github.com/didrikolofsson/github-vote-llm/internal/oauth2"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/google/go-github/v68/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	oa "golang.org/x/oauth2"
)

var ErrGitHubNotConnected = errors.New("github: no connection found for user")

type GithubService interface {
	Callback(ctx context.Context, code string, userID int64, tokenEncryptionKey string) error
	Status(ctx context.Context, userID int64) (*GithubUserResponse, error)
	ListReposByAuthenticatedUser(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error)
}

type GithubServiceImpl struct {
	db                 *pgxpool.Pool
	q                  *store.Queries
	config             *oa.Config
	tokenEncryptionKey string
}

func NewGithubService(db *pgxpool.Pool, q *store.Queries, config *oa.Config, tokenEncryptionKey string) GithubService {
	return &GithubServiceImpl{db: db, q: q, config: config, tokenEncryptionKey: tokenEncryptionKey}
}

func (s *GithubServiceImpl) Callback(ctx context.Context, code string, userID int64, tokenEncryptionKey string) error {
	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		return err
	}
	encryptedAccessTokenBytes, err := encryption.Encrypt([]byte(token.AccessToken), tokenEncryptionKey)
	if err != nil {
		return err
	}
	encryptedAccessToken := base64.StdEncoding.EncodeToString(encryptedAccessTokenBytes)
	var encryptedRefreshToken *string
	if token.RefreshToken != "" {
		encryptedRefreshBytes, encErr := encryption.Encrypt([]byte(token.RefreshToken), tokenEncryptionKey)
		if encErr != nil {
			return encErr
		}
		encoded := base64.StdEncoding.EncodeToString(encryptedRefreshBytes)
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

	ts := oauth2.NewGithubTokenSource(userID, s.q, s.config, s.tokenEncryptionKey)
	httpClient := oa.NewClient(ctx, ts)
	githubClient := github.NewClient(httpClient)
	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return &GithubUserResponse{
		ID:    *user.ID,
		Login: *user.Login,
	}, nil
}

func (s *GithubServiceImpl) ListReposByAuthenticatedUser(ctx context.Context, userID int64, page int) ([]dtos.GitHubRepository, bool, error) {
	if _, err := s.q.GetGitHubConnectionByUserID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrGitHubNotConnected
		}
		return nil, false, err
	}

	ts := oauth2.NewGithubTokenSource(userID, s.q, s.config, s.tokenEncryptionKey)
	httpClient := oa.NewClient(ctx, ts)
	githubClient := github.NewClient(httpClient)

	repos, resp, err := githubClient.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
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
