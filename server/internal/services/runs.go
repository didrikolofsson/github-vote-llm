package services

import (
	"context"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	api_errors "github.com/didrikolofsson/github-vote-llm/internal/errors"
	"github.com/didrikolofsson/github-vote-llm/internal/jobs/args"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type CreateRunParams struct {
	Prompt    string
	FeatureID int64
	UserID    int64
}

type RunService interface {
	CreateRun(ctx context.Context, p CreateRunParams) (*dtos.RunDTO, error)
}

type RunServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
	jc *river.Client[pgx.Tx]
}

func NewRunService(db *pgxpool.Pool, q *store.Queries, jc *river.Client[pgx.Tx]) RunService {
	return &RunServiceImpl{db: db, q: q, jc: jc}
}

func storeToRunDTO(run store.FeatureRun) *dtos.RunDTO {
	return &dtos.RunDTO{
		ID:        run.ID,
		Prompt:    run.Prompt,
		FeatureID: run.FeatureID,
		Status:    dtos.RunStatus(run.Status),
	}
}

func (s *RunServiceImpl) CreateRun(ctx context.Context, p CreateRunParams) (*dtos.RunDTO, error) {
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
	repo, err := qtx.GetRepositoryByFeatureID(ctx, p.FeatureID)
	if err != nil {
		return nil, err
	}

	if _, err := s.jc.InsertTx(ctx, tx, args.CloneRepoArgs{
		UserID: p.UserID,
		RunID:  run.ID,
		Owner:  repo.Owner,
		Name:   repo.Name,
	}, nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return storeToRunDTO(run), nil
}
