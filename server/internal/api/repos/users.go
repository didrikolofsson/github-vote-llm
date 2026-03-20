package repos

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	CreateUser(ctx context.Context, params store.CreateUserParams) (*store.CreateUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (*store.GetUserByEmailRow, error)
	GetUserByID(ctx context.Context, id int64) (*store.GetUserByIDRow, error)
	DeleteUser(ctx context.Context, id int64) error
}

type UserRepositoryImpl struct {
	q *store.Queries
}

func NewUserRepository(db *pgx.Conn) UserRepository {
	q := store.New(db)
	return &UserRepositoryImpl{q: q}
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, params store.CreateUserParams) (*store.CreateUserRow, error) {
	user, err := r.q.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id int64) (*store.GetUserByIDRow, error) {
	user, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*store.GetUserByEmailRow, error) {
	user, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, id int64) error {
	err := r.q.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	return nil
}
