package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/api/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateRunBody struct {
	Prompt          string `json:"prompt"`
	CreatedByUserID int64  `json:"created_by_user_id"`
}

type CreateRunParams struct {
	Prompt    string
	FeatureID int64
	UserID    int64
	Env       *config.Environment
	ApiKey    string
}

type RunService interface {
	CreateRun(ctx context.Context, p CreateRunParams) (*dtos.RunDTO, error)
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

// Create new run db record
func (s *RunServiceImpl) CreateRun(
	ctx context.Context,
	p CreateRunParams,
) (*dtos.RunDTO, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	dto := storeToRunDTO(run)
	return dto, nil
}
