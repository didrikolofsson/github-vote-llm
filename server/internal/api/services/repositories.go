package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryAlreadyAdded = errors.New("repository already added to organization")
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrNotOrgMember           = errors.New("not a member of this organization")
)

type Repository struct {
	ID        int64  `json:"id"`
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
}

type RepositoriesService interface {
	ListForOrganization(ctx context.Context, orgID, userID int64) ([]Repository, error)
	GetRepository(ctx context.Context, repoID, userID int64) (*Repository, error)
	AddRepository(ctx context.Context, orgID, userID int64, owner, name string) (*Repository, error)
	RemoveRepository(ctx context.Context, repoID, userID int64) error
}

type RepositoriesServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewRepositoriesService(db *pgxpool.Pool, q *store.Queries) RepositoriesService {
	return &RepositoriesServiceImpl{db: db, q: q}
}

func (s *RepositoriesServiceImpl) ListForOrganization(ctx context.Context, orgID, userID int64) ([]Repository, error) {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListRepositoriesForOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Repository, len(rows))
	for i, r := range rows {
		out[i] = storeRepoToDTO(r)
	}
	return out, nil
}

func (s *RepositoriesServiceImpl) GetRepository(ctx context.Context, repoID, userID int64) (*Repository, error) {
	r, err := s.q.GetRepository(ctx, repoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.verifyOrgMember(ctx, r.OrganizationID, userID); err != nil {
		return nil, err
	}
	dto := storeRepoToDTO(r)
	return &dto, nil
}

func (s *RepositoriesServiceImpl) AddRepository(ctx context.Context, orgID, userID int64, owner, name string) (*Repository, error) {
	if err := s.verifyOrgMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	r, err := s.q.AddRepository(ctx, store.AddRepositoryParams{
		OrganizationID: orgID,
		Owner:          owner,
		Name:           name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrRepositoryAlreadyAdded
		}
		return nil, err
	}
	dto := storeRepoToDTO(r)
	return &dto, nil
}

func (s *RepositoriesServiceImpl) RemoveRepository(ctx context.Context, repoID, userID int64) error {
	r, err := s.q.GetRepository(ctx, repoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if err := s.verifyOrgMember(ctx, r.OrganizationID, userID); err != nil {
		return err
	}
	return s.q.RemoveRepository(ctx, repoID)
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

func storeRepoToDTO(r store.Repository) Repository {
	return Repository{
		ID:        r.ID,
		Owner:     r.Owner,
		Name:      r.Name,
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
