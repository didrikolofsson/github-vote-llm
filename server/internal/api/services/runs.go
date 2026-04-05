package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/api/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/api/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/river/jobs"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
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
	rc *river.Client[pgx.Tx]
}

func NewRunService(db *pgxpool.Pool, q *store.Queries, rc *river.Client[pgx.Tx]) RunService {
	return &RunServiceImpl{db: db, q: q, rc: rc}
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
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	run, err := qtx.CreateRun(ctx, store.CreateRunParams{
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

	_, err = s.rc.InsertTx(ctx, tx, &jobs.RunClaudeArgs{
		Prompt:    prompt,
		FeatureID: featureID,
		UserID:    createdByUserID,
	}, nil)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	dto := storeToRunDTO(run)
	return dto, nil
}
