package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrMemberNotFound      = errors.New("member not found")
	ErrInviteUserNotFound  = errors.New("user not found")
	ErrUserAlreadyInOrg    = errors.New("user is already in an organization")
	ErrCannotRemoveOwner   = errors.New("cannot remove the last owner")
	ErrCannotChangeOwnRole = errors.New("owners cannot change their own role")
)

type MemberWithEmail struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type MembersService struct {
	q *store.Queries
}

func NewMembersService(q *store.Queries) *MembersService {
	return &MembersService{q: q}
}

func (s *MembersService) ListMembers(ctx context.Context, orgID, requestingUserID int64) ([]MemberWithEmail, error) {
	if err := s.verifyOrgMember(ctx, orgID, requestingUserID); err != nil {
		return nil, err
	}
	rows, err := s.q.GetOrganizationMembersWithUser(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberWithEmail, len(rows))
	for i, r := range rows {
		out[i] = MemberWithEmail{UserID: r.UserID, Email: r.Email, Role: string(r.Role)}
	}
	return out, nil
}

func (s *MembersService) InviteByEmail(ctx context.Context, orgID, requestingUserID int64, email string) error {
	if err := s.verifyOrgOwner(ctx, orgID, requestingUserID); err != nil {
		return err
	}

	user, err := s.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInviteUserNotFound
	}
	if err != nil {
		return err
	}

	_, err = s.q.AddOrganizationMember(ctx, store.AddOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         user.ID,
		Role:           store.OrganizationMemberRoleMember,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserAlreadyInOrg
		}
		return err
	}
	return nil
}

func (s *MembersService) RemoveMember(ctx context.Context, orgID, requestingUserID int64, userID int64) error {
	if err := s.verifyOrgOwner(ctx, orgID, requestingUserID); err != nil {
		return err
	}

	members, err := s.q.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	var targetIsOwner bool
	var ownerCount int
	for _, m := range members {
		if m.Role == store.OrganizationMemberRoleOwner {
			ownerCount++
		}
		if m.UserID == userID {
			targetIsOwner = m.Role == store.OrganizationMemberRoleOwner
		}
	}
	if targetIsOwner && ownerCount <= 1 {
		return ErrCannotRemoveOwner
	}

	err = s.q.RemoveOrganizationMember(ctx, store.RemoveOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *MembersService) UpdateRole(ctx context.Context, orgID, requestingUserID int64, userID int64, role string) error {
	if err := s.verifyOrgOwner(ctx, orgID, requestingUserID); err != nil {
		return err
	}
	if requestingUserID == userID {
		return ErrCannotChangeOwnRole
	}

	roleEnum := store.OrganizationMemberRole(role)
	if roleEnum != store.OrganizationMemberRoleOwner && roleEnum != store.OrganizationMemberRoleMember {
		return errors.New("invalid role")
	}

	members, err := s.q.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	var targetIsOwner bool
	var ownerCount int
	for _, m := range members {
		if m.Role == store.OrganizationMemberRoleOwner {
			ownerCount++
		}
		if m.UserID == userID {
			targetIsOwner = m.Role == store.OrganizationMemberRoleOwner
		}
	}
	// Downgrading last owner
	if targetIsOwner && ownerCount <= 1 && roleEnum == store.OrganizationMemberRoleMember {
		return ErrCannotRemoveOwner
	}

	_, err = s.q.UpdateOrganizationMemberRole(ctx, store.UpdateOrganizationMemberRoleParams{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           roleEnum,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMemberNotFound
	}
	return err
}

func (s *MembersService) verifyOrgMember(ctx context.Context, orgID, userID int64) error {
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

func (s *MembersService) verifyOrgOwner(ctx context.Context, orgID, userID int64) error {
	members, err := s.q.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.UserID == userID && m.Role == store.OrganizationMemberRoleOwner {
			return nil
		}
	}
	return ErrNotOrgMember
}
