package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryAlreadyAdded  = errors.New("repository already added to organization")
	ErrRepositoryNotFound      = errors.New("repository not found")
	ErrRepositoryNotAccessible = errors.New("repository not visible to the organization's GitHub app")
)

type RepoServiceDeps struct {
	DB        *pgxpool.Pool
	Queries   *store.Queries
	AppClient *github.AppClient
}

type RepoService struct {
	db        *pgxpool.Pool
	q         *store.Queries
	appClient *github.AppClient
}

func NewRepoService(deps RepoServiceDeps) *RepoService {
	return &RepoService{db: deps.DB, q: deps.Queries, appClient: deps.AppClient}
}

func (s *RepoService) ListForOrganization(ctx context.Context, orgID, userID int64) ([]dtos.Repository, error) {
	rows, err := s.q.ListRepositoriesForOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]dtos.Repository, len(rows))
	for i, r := range rows {
		out[i] = storeRepoToDTO(r)
	}
	return out, nil
}

func (s *RepoService) GetRepositoryByID(ctx context.Context, repoID, userID int64) (*dtos.Repository, error) {
	r, err := s.q.GetRepository(ctx, repoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	dto := storeRepoToDTO(r)
	return &dto, nil
}

func (s *RepoService) GetRepositoryByOwnerAndName(ctx context.Context, orgID int64, owner, name string) (*dtos.Repository, error) {
	r, err := s.q.GetRepositoryByOwnerAndName(ctx, store.GetRepositoryByOwnerAndNameParams{
		OrganizationID: orgID,
		Owner:          owner,
		Name:           name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	dto := storeRepoToDTO(r)
	return &dto, nil
}

func (s *RepoService) GetRepositoryMeta(ctx context.Context, repoID int64) (*dtos.RepoMeta, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := s.q.WithTx(tx)

	description := ""
	repo, err := qtx.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo.Description != nil {
		description = *repo.Description
	}

	featureCount, err := qtx.GetRepositoryFeatureCount(ctx, repoID)
	if err != nil {
		return nil, err
	}
	return &dtos.RepoMeta{
		ID:              repoID,
		Description:     description,
		Features:        featureCount,
		Implementations: 0,
		Status:          dtos.RepoStatus("idle"),
	}, nil

}

func (s *RepoService) AddRepository(ctx context.Context, orgID, userID int64, owner, name string) (*dtos.Repository, error) {
	installation, err := s.q.GetInstallationByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInstallationNotFound
		}
		return nil, err
	}
	if installation.SuspendedAt.Valid {
		return nil, ErrInstallationSuspended
	}

	client, err := s.appClient.InstallationClient(ctx, installation.GithubInstallationID)
	if err != nil {
		return nil, err
	}

	_, resp, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRepositoryNotAccessible
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

func (s *RepoService) UpdatePortalPublic(ctx context.Context, repoID, userID int64, public bool) (*dtos.Repository, error) {
	_, err := s.q.GetRepository(ctx, repoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	updated, err := s.q.SetRepositoryPortalPublic(ctx, store.SetRepositoryPortalPublicParams{
		ID:           repoID,
		PortalPublic: public,
	})
	if err != nil {
		return nil, err
	}
	dto := storeRepoToDTO(updated)
	return &dto, nil
}

func (s *RepoService) RemoveRepository(ctx context.Context, repoID, userID int64) error {
	_, err := s.q.GetRepository(ctx, repoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	return s.q.RemoveRepository(ctx, repoID)
}

func storeRepoToDTO(r store.OrganizationRepository) dtos.Repository {
	return dtos.Repository{
		ID:           r.ID,
		Owner:        r.Owner,
		Name:         r.Name,
		PortalPublic: r.PortalPublic,
		CreatedAt:    r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
