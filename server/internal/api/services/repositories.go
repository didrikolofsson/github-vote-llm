package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrRepositoryAlreadyAdded = errors.New("repository already added to organization")
	ErrRepositoryNotFound    = errors.New("repository not found in organization")
	ErrGitHubNotConnected     = errors.New("connect GitHub first to add repositories")
	ErrNotOrgMember          = errors.New("not a member of this organization")
)

type Repository struct {
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	CreatedAt string `json:"created_at,omitempty"`
}

type RepositoriesService interface {
	ListForOrganization(ctx context.Context, orgID, userID int64) ([]Repository, error)
	AddRepository(ctx context.Context, orgID, userID int64, owner, repo string) error
	RemoveRepository(ctx context.Context, orgID, userID int64, owner, repo string) error
	ListAvailableFromGitHub(ctx context.Context, orgID, userID int64, page int) ([]Repository, bool, error)
}

type RepositoriesServiceImpl struct {
	q             *store.Queries
	githubOAuth   GitHubOAuthService
	encryptionKey string
}

func NewRepositoriesService(q *store.Queries, githubOAuth GitHubOAuthService, encryptionKey string) RepositoriesService {
	return &RepositoriesServiceImpl{
		q:             q,
		githubOAuth:   githubOAuth,
		encryptionKey: encryptionKey,
	}
}

func (s *RepositoriesServiceImpl) ListForOrganization(ctx context.Context, orgID int64, userID int64) ([]Repository, error) {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListRepositoriesForOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Repository, len(rows))
	for i, r := range rows {
		out[i] = Repository{
			Owner:     r.Owner,
			Repo:      r.Repo,
			CreatedAt: r.CreatedAt.Time.Format("2006-01-02"),
		}
	}
	return out, nil
}

func (s *RepositoriesServiceImpl) AddRepository(ctx context.Context, orgID, userID int64, owner, repo string) error {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return err
	}

	_, err := s.q.AddOrganizationRepository(ctx, store.AddOrganizationRepositoryParams{
		OrganizationID: orgID,
		Owner:          owner,
		Repo:           repo,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrRepositoryAlreadyAdded
		}
		return err
	}
	return nil
}

func (s *RepositoriesServiceImpl) RemoveRepository(ctx context.Context, orgID, userID int64, owner, repo string) error {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return err
	}
	_, err := s.q.GetOrganizationRepository(ctx, store.GetOrganizationRepositoryParams{
		OrganizationID: orgID,
		Owner:          owner,
		Repo:           repo,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	return s.q.RemoveOrganizationRepository(ctx, store.RemoveOrganizationRepositoryParams{
		OrganizationID: orgID,
		Owner:          owner,
		Repo:           repo,
	})
}

func (s *RepositoriesServiceImpl) verifyOrgMember(ctx context.Context, orgID, userID int64) error {
	members, err := s.q.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.UserID == userID {
			return nil
		}
	}
	return ErrNotOrgMember
}

func (s *RepositoriesServiceImpl) ListAvailableFromGitHub(ctx context.Context, orgID, userID int64, page int) ([]Repository, bool, error) {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return nil, false, err
	}
	token, err := s.githubOAuth.GetDecryptedToken(ctx, userID, s.encryptionKey)
	if err != nil || token == "" {
		return nil, false, ErrGitHubNotConnected
	}

	client := github.NewOAuthClient(token)
	summaries, hasMore, err := client.ListRepos(ctx, page)
	if err != nil {
		return nil, false, err
	}

	out := make([]Repository, len(summaries))
	for i, r := range summaries {
		out[i] = Repository{Owner: r.Owner, Repo: r.Repo}
	}
	return out, hasMore, nil
}
