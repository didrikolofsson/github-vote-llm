package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateRunParams struct {
	Prompt    string
	FeatureID int64
	UserID    int64
}

type RunService interface {
	CreateRun(ctx context.Context, tx pgx.Tx, p CreateRunParams) (*dtos.RunDTO, error)
}

type RunServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewRunService(db *pgxpool.Pool, q *store.Queries) RunService {
	return &RunServiceImpl{db: db, q: q}
}

func storeToRunDTO(run store.FeatureRun) *dtos.RunDTO {
	return &dtos.RunDTO{
		ID:        run.ID,
		Prompt:    run.Prompt,
		FeatureID: run.FeatureID,
		Status:    dtos.RunStatus(run.Status),
	}
}

func (s *RunServiceImpl) CreateRun(ctx context.Context, tx pgx.Tx, p CreateRunParams) (*dtos.RunDTO, error) {
	qtx := s.q.WithTx(tx)
	run, err := qtx.CreateRun(ctx, store.CreateRunParams{
		Prompt:          p.Prompt,
		FeatureID:       p.FeatureID,
		Status:          store.FeatureRunStatusPending,
		CreatedByUserID: p.UserID,
	})
	if err != nil {
		if api_errors.IsForeignKeyViolationErr(err) {
			return nil, ErrFeatureNotFound
		}
		return nil, err
	}
	return storeToRunDTO(run), nil
}
