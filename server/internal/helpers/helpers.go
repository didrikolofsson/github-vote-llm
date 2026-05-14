package helpers

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// Float64ToNumeric converts *float64 to pgtype.Numeric.
func Float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*f); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

var (
	ErrNotOrgMember = errors.New("not a member of this organization")
	ErrNotOrgOwner  = errors.New("not the owner of this organization")
)

func VerifyOrgMember(ctx context.Context, q *store.Queries, orgID, userID int64) error {
	isMember, err := q.IsOrgMember(ctx, store.IsOrgMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotOrgMember
	}
	return nil
}

func VerifyOrgOwner(ctx context.Context, q *store.Queries, orgID, userID int64) error {
	members, err := q.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.UserID == userID && m.Role == store.OrganizationMemberRoleOwner {
			return nil
		}
	}
	return ErrNotOrgOwner
}
