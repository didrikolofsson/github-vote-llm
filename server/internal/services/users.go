package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUsernameTaken = errors.New("username already taken")

type UserService struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewUserService(db *pgxpool.Pool, q *store.Queries) *UserService {
	return &UserService{db: db, q: q}
}

var (
	ErrUserExists          = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrForbiddenUserDelete = errors.New("forbidden: cannot delete this user")
)

func (s *UserService) SignupUser(ctx context.Context, params *store.CreateUserParams) (*dtos.User, error) {
	hashedPassword, err := helpers.HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.q.CreateUser(ctx, store.CreateUserParams{
		Email:    params.Email,
		Password: hashedPassword,
	})

	if err != nil {
		if api_errors.IsAlreadyExistsErr(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return &dtos.User{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (*dtos.User, error) {
	row, err := s.q.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return userRowToDTO(row), nil
}

func (s *UserService) UpdateUsername(ctx context.Context, userID int64, username string) (*dtos.User, error) {
	row, err := s.q.UpdateUserUsername(ctx, store.UpdateUserUsernameParams{
		ID:       userID,
		Username: &username,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	return &dtos.User{
		ID:        row.ID,
		Email:     row.Email,
		Username:  row.Username,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// prepareAccountDeletion handles the organization_members trigger that forbids
// removing the last owner: delete the org when the user is the only member, or
// promote another member to owner before the user row is removed.
func (s *UserService) prepareAccountDeletion(ctx context.Context, qtx *store.Queries, userID int64) error {
	membership, err := qtx.GetOrganizationMembershipByUserID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	members, err := qtx.GetOrganizationMembers(ctx, membership.OrganizationID)
	if err != nil {
		return err
	}
	if membership.Role != store.OrganizationMemberRoleOwner {
		return nil
	}

	ownerCount := 0
	for _, m := range members {
		if m.Role == store.OrganizationMemberRoleOwner {
			ownerCount++
		}
	}
	if ownerCount != 1 {
		return nil
	}

	if len(members) == 1 {
		return qtx.DeleteOrganization(ctx, membership.OrganizationID)
	}

	var promoteID int64
	for _, m := range members {
		if m.UserID == userID {
			continue
		}
		if promoteID == 0 || m.UserID < promoteID {
			promoteID = m.UserID
		}
	}
	if promoteID == 0 {
		return errors.New("could not pick new owner for organization")
	}

	_, err = qtx.UpdateOrganizationMemberRole(ctx, store.UpdateOrganizationMemberRoleParams{
		OrganizationID: membership.OrganizationID,
		UserID:         promoteID,
		Role:           store.OrganizationMemberRoleOwner,
	})
	return err
}

// verifyUserDeletionAllowed permits deleting targetUserID when the requester is the same user,
// or when the requester is an owner of the organization that targetUserID belongs to.
func verifyUserDeletionAllowed(ctx context.Context, qtx *store.Queries, requesterID, targetID int64) error {
	if requesterID == targetID {
		return nil
	}
	reqMem, err := qtx.GetOrganizationMembershipByUserID(ctx, requesterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbiddenUserDelete
	}
	if err != nil {
		return err
	}
	if reqMem.Role != store.OrganizationMemberRoleOwner {
		return ErrForbiddenUserDelete
	}
	tgtMem, err := qtx.GetOrganizationMembershipByUserID(ctx, targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbiddenUserDelete
	}
	if err != nil {
		return err
	}
	if reqMem.OrganizationID != tgtMem.OrganizationID {
		return ErrForbiddenUserDelete
	}
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, requestingUserID, targetUserID int64) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	user, err := qtx.GetUserByID(ctx, targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if err := verifyUserDeletionAllowed(ctx, qtx, requestingUserID, targetUserID); err != nil {
		return err
	}
	if err := s.prepareAccountDeletion(ctx, qtx, targetUserID); err != nil {
		return err
	}
	if err := qtx.DeleteUser(ctx, user.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func userRowToDTO(row store.GetUserByIDRow) *dtos.User {
	return &dtos.User{
		ID:        row.ID,
		Email:     row.Email,
		Username:  row.Username,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
