package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/api/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateRunBody struct {
	Prompt          string `json:"prompt"`
	CreatedByUserID int64  `json:"created_by_user_id"`
}

type RunService interface {
	CreateRun(ctx context.Context, prompt string, featureID, createdByUserID int64) (*dtos.RunDTO, error)
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
	prompt string,
	featureID,
	createdByUserID int64,
) (*dtos.RunDTO, error) {
	run, err := s.q.CreateRun(ctx, store.CreateRunParams{
		Prompt:          prompt,
		FeatureID:       featureID,
		Status:          store.FeatureRunStatusPending,
		CreatedByUserID: createdByUserID,
	})

	if err != nil {
		if api_errors.IsForeignKeyViolationErr(err) {
			return nil, ErrFeatureNotFound
		}
		return nil, err
	}
	dto := storeToRunDTO(run)
	return dto, nil
}
