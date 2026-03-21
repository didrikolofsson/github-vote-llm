package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserService interface {
	SignupUser(ctx context.Context, params *store.CreateUserParams) (*dtos.User, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type UserServiceImpl struct {
	db *pgx.Conn
	q  *store.Queries
}

func NewUserService(db *pgx.Conn, q *store.Queries) UserService {
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
