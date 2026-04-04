package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateRunParams struct {
	Prompt          string `json:"promp"`
	FeatureID       int64  `json:"feature_id"`
	CreatedByUserID int64  `json:"created_by_user_id"`
}

type RunService interface {
	CreateRun(ctx context.Context, params CreateRunParams) (*dtos.RunDTO, error)
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
func (s *RunServiceImpl) CreateRun(ctx context.Context, params CreateRunParams) (*dtos.RunDTO, error) {

	run, err := s.q.CreateRun(ctx, store.CreateRunParams{
		Prompt:          params.Prompt,
		FeatureID:       params.FeatureID,
		Status:          store.FeatureRunStatusPending,
		CreatedByUserID: params.CreatedByUserID,
	})

	if err != nil {
		return nil, err
	}
	dto := storeToRunDTO(run)
	return dto, nil
}
