package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrOrganizationNotFound   = errors.New("organization not found")
	ErrOrganizationNameExists = errors.New("organization name already exists")
)

type CreateOrganizationParams struct {
	Name    string
	OwnerID int64
}

type OrganizationService interface {
	CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.OrganizationWithMembers, error)
	GetOrganizationByID(ctx context.Context, organizationID int64) (*dtos.OrganizationWithMembers, error)
	UpdateOrganizationByID(ctx context.Context, organizationID int64, params *store.UpdateOrganizationByIDParams) (*dtos.Organization, error)
	DeleteOrganization(ctx context.Context, organizationID int64) error
}

type OrganizationServiceImpl struct {
	db *pgx.Conn
	q  *store.Queries
}

func NewOrganizationService(db *pgx.Conn, q *store.Queries) OrganizationService {
	return &OrganizationServiceImpl{db: db, q: q}
}

func (s *OrganizationServiceImpl) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.OrganizationWithMembers, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	org, err := qtx.CreateOrganization(ctx, params.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrOrganizationNameExists
		}
		return nil, err
	}

	_, err = qtx.AddOrganizationMember(ctx, store.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         params.OwnerID,
		Role:           store.OrganizationMemberRoleOwner,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	members, _ := s.q.GetOrganizationMembers(ctx, org.ID)
	return &dtos.OrganizationWithMembers{
		Organization: storeOrgToDTO(org),
		Members:      storeMembersToDTOs(members),
	}, nil
}

func (s *OrganizationServiceImpl) GetOrganizationByID(ctx context.Context, organizationID int64) (*dtos.OrganizationWithMembers, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	org, err := qtx.GetOrganizationByID(ctx, organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}

	members, err := qtx.GetOrganizationMembers(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dtos.OrganizationWithMembers{
		Organization: storeOrgToDTO(org),
		Members:      storeMembersToDTOs(members),
	}, nil
}

func (s *OrganizationServiceImpl) UpdateOrganizationByID(
	ctx context.Context,
	organizationID int64,
	params *store.UpdateOrganizationByIDParams,
) (*dtos.Organization, error) {
	arg := store.UpdateOrganizationByIDParams{ID: organizationID, Name: params.Name}
	org, err := s.q.UpdateOrganizationByID(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrganizationNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrOrganizationNameExists
		}
		return nil, err
	}
	return storeOrgToDTOPtr(org), nil
}

func (s *OrganizationServiceImpl) DeleteOrganization(ctx context.Context, organizationID int64) error {
	return s.q.DeleteOrganization(ctx, organizationID)
}

func storeOrgToDTO(o store.Organization) dtos.Organization {
	return dtos.Organization{
		ID:        o.ID,
		Name:      o.Name,
		CreatedAt: o.CreatedAt.Time,
		UpdatedAt: o.UpdatedAt.Time,
	}
}

func storeOrgToDTOPtr(o store.Organization) *dtos.Organization {
	d := storeOrgToDTO(o)
	return &d
}

func storeMembersToDTOs(members []store.OrganizationMember) []dtos.OrganizationMember {
	out := make([]dtos.OrganizationMember, len(members))
	for i, m := range members {
		out[i] = dtos.OrganizationMember{
			UserID: m.UserID,
			Role:   string(m.Role),
		}
	}
	return out
}
