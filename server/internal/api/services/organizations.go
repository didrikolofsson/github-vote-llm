package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
)

type CreateOrganizationParams struct {
	Name    string
	OwnerID int64
}

type OrganizationService interface {
	CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.Organization, error)
	DeleteOrganization(ctx context.Context, organizationID int64) error
}

type OrganizationServiceImpl struct {
	db *pgx.Conn
	q  *store.Queries
}

func NewOrganizationService(db *pgx.Conn, q *store.Queries) OrganizationService {
	return &OrganizationServiceImpl{db: db, q: q}
}

func (s *OrganizationServiceImpl) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.Organization, error) {
	return nil, nil
}

func (s *OrganizationServiceImpl) DeleteOrganization(ctx context.Context, organizationID int64) error {
	return nil
}
