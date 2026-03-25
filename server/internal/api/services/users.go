package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUsernameTaken = errors.New("username already taken")

type UserService interface {
	SignupUser(ctx context.Context, params *store.CreateUserParams) (*dtos.User, error)
	DeleteUser(ctx context.Context, userID int64) error
	GetUser(ctx context.Context, userID int64) (*dtos.User, error)
	UpdateUsername(ctx context.Context, userID int64, username string) (*dtos.User, error)
}

type UserServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewUserService(db *pgxpool.Pool, q *store.Queries) UserService {
	return &UserServiceImpl{db: db, q: q}
}

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

func (s *UserServiceImpl) SignupUser(ctx context.Context, params *store.CreateUserParams) (*dtos.User, error) {
	hashedPassword, err := helpers.HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.q.CreateUser(ctx, store.CreateUserParams{
		Email:    params.Email,
		Password: hashedPassword,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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

func (s *UserServiceImpl) GetUser(ctx context.Context, userID int64) (*dtos.User, error) {
	row, err := s.q.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return userRowToDTO(row), nil
}

func (s *UserServiceImpl) UpdateUsername(ctx context.Context, userID int64, username string) (*dtos.User, error) {
	row, err := s.q.UpdateUserUsername(ctx, store.UpdateUserUsernameParams{
		ID:       userID,
		Username: pgtype.Text{String: username, Valid: true},
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
		Username:  nullableString(row.Username),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, userID int64) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	user, err := qtx.GetUserByID(ctx, userID)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
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
		Username:  nullableString(row.Username),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func nullableString(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
