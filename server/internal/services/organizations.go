package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrganizationNotFound      = errors.New("organization not found")
	ErrOrganizationNameExists    = errors.New("organization name already exists")
	ErrOrganizationSlugExists    = errors.New("organization slug already exists")
	ErrUserAlreadyInOrganization = errors.New("you already belong to an organization")

	nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)
)

// slugify converts a human-readable name into a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

type CreateOrganizationParams struct {
	Name    string
	Slug    string // optional; derived from Name when empty
	OwnerID int64
}

type OrganizationService interface {
	CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.OrganizationWithMembers, error)
	GetOrganizationByID(ctx context.Context, organizationID int64) (*dtos.OrganizationWithMembers, error)
	ListOrganizationsForUser(ctx context.Context, userID int64) ([]dtos.Organization, error)
	UpdateOrganizationByID(ctx context.Context, organizationID int64, params *store.UpdateOrganizationByIDParams) (*dtos.Organization, error)
	UpdateOrganizationSlug(ctx context.Context, organizationID int64, slug string) (*dtos.Organization, error)
	DeleteOrganization(ctx context.Context, organizationID int64) error
}

type OrganizationServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewOrganizationService(db *pgxpool.Pool, q *store.Queries) OrganizationService {
	return &OrganizationServiceImpl{db: db, q: q}
}

func (s *OrganizationServiceImpl) ListOrganizationsForUser(ctx context.Context, userID int64) ([]dtos.Organization, error) {
	orgs, err := s.q.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]dtos.Organization, len(orgs))
	for i, o := range orgs {
		out[i] = makeOrgDTO(o.ID, o.Name, o.Slug, o.CreatedAt, o.UpdatedAt)
	}
	return out, nil
}

func (s *OrganizationServiceImpl) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*dtos.OrganizationWithMembers, error) {
	existing, _ := s.q.ListOrganizationsForUser(ctx, params.OwnerID)
	if len(existing) > 0 {
		return nil, ErrUserAlreadyInOrganization
	}

	slug := params.Slug
	if slug == "" {
		slug = slugify(params.Name)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	org, err := qtx.CreateOrganization(ctx, store.CreateOrganizationParams{
		Name: params.Name,
		Slug: slug,
	})
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

	members, _ := s.q.GetOrganizationMembersWithUser(ctx, org.ID)
	return &dtos.OrganizationWithMembers{
		Organization: makeOrgDTO(org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt),
		Members:      storeMembersWithEmailToDTOs(members),
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

	members, err := qtx.GetOrganizationMembersWithUser(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dtos.OrganizationWithMembers{
		Organization: makeOrgDTO(org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt),
		Members:      storeMembersWithEmailToDTOs(members),
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
	dto := makeOrgDTO(org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	return &dto, nil
}

func (s *OrganizationServiceImpl) UpdateOrganizationSlug(ctx context.Context, organizationID int64, slug string) (*dtos.Organization, error) {
	org, err := s.q.UpdateOrganizationSlug(ctx, store.UpdateOrganizationSlugParams{
		ID:   organizationID,
		Slug: slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrganizationNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrOrganizationSlugExists
		}
		return nil, err
	}
	dto := makeOrgDTO(org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	return &dto, nil
}

func (s *OrganizationServiceImpl) DeleteOrganization(ctx context.Context, organizationID int64) error {
	return s.q.DeleteOrganization(ctx, organizationID)
}

// makeOrgDTO converts raw organization fields to the API DTO.
func makeOrgDTO(id int64, name, slug string, createdAt, updatedAt pgtype.Timestamptz) dtos.Organization {
	return dtos.Organization{
		ID:        id,
		Name:      name,
		Slug:      slug,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}
}

func storeMembersWithEmailToDTOs(members []store.GetOrganizationMembersWithUserRow) []dtos.OrganizationMember {
	out := make([]dtos.OrganizationMember, len(members))
	for i, m := range members {
		out[i] = dtos.OrganizationMember{
			UserID: m.UserID,
			Email:  m.Email,
			Role:   string(m.Role),
		}
	}
	return out
}
