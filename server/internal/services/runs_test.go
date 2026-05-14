package services

import (
	"context"
	"errors"
	"testing"

	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestRunService(t *testing.T) (*RunService, *store.MockQuerier, *hub.MockHub) {
	t.Helper()
	ctrl := gomock.NewController(t)
	q := store.NewMockQuerier(ctrl)
	h := hub.NewMockHub(ctrl)
	svc := &RunService{
		db:     nil,
		q:      q,
		withTx: nil, // not needed for CancelRun / DeleteRun tests
		hub:    h,
		logHub: hub.NewRunLogHub(),
	}
	return svc, q, h
}

func pendingRun(repoID int64) store.GetRunByIDRow {
	return store.GetRunByIDRow{
		ID:           1,
		RepositoryID: repoID,
		Status:       store.FeatureRunStatusPending,
	}
}

func runningRun(repoID int64, pid int32) store.GetRunByIDRow {
	return store.GetRunByIDRow{
		ID:           2,
		RepositoryID: repoID,
		Status:       store.FeatureRunStatusRunning,
		Pid:          &pid,
	}
}

func completedRun(repoID int64) store.GetRunByIDRow {
	return store.GetRunByIDRow{
		ID:           3,
		RepositoryID: repoID,
		Status:       store.FeatureRunStatusCompleted,
	}
}

func TestCancelRun_Pending(t *testing.T) {
	svc, q, h := newTestRunService(t)
	ctx := context.Background()

	const repoID int64 = 10
	run := pendingRun(repoID)

	q.EXPECT().GetRunByID(ctx, run.ID).Return(run, nil)
	q.EXPECT().SetRunCancelled(ctx, run.ID).Return(nil)
	h.EXPECT().Publish(repoID, hub.EventRunUpdated)

	err := svc.CancelRun(ctx, run.ID)
	require.NoError(t, err)
}

func TestCancelRun_Running(t *testing.T) {
	svc, q, h := newTestRunService(t)
	ctx := context.Background()

	const repoID int64 = 10
	var pid int32 = 99999 // high PID unlikely to exist — kill will fail silently
	run := runningRun(repoID, pid)

	q.EXPECT().GetRunByID(ctx, run.ID).Return(run, nil)
	q.EXPECT().SetRunCancelled(ctx, run.ID).Return(nil)
	h.EXPECT().Publish(repoID, hub.EventRunUpdated)

	err := svc.CancelRun(ctx, run.ID)
	require.NoError(t, err)
}

func TestCancelRun_NotCancellable(t *testing.T) {
	svc, q, _ := newTestRunService(t)
	ctx := context.Background()

	run := completedRun(10)

	q.EXPECT().GetRunByID(ctx, run.ID).Return(run, nil)

	err := svc.CancelRun(ctx, run.ID)
	assert.ErrorIs(t, err, ErrRunNotCancellable)
}

func TestCancelRun_NotFound(t *testing.T) {
	svc, q, _ := newTestRunService(t)
	ctx := context.Background()

	q.EXPECT().GetRunByID(ctx, int64(99)).Return(store.GetRunByIDRow{}, pgx.ErrNoRows)

	err := svc.CancelRun(ctx, 99)
	assert.ErrorIs(t, err, ErrRunNotFound)
}

func TestCancelRun_DBError(t *testing.T) {
	svc, q, _ := newTestRunService(t)
	ctx := context.Background()

	dbErr := errors.New("db down")
	q.EXPECT().GetRunByID(ctx, int64(1)).Return(store.GetRunByIDRow{}, dbErr)

	err := svc.CancelRun(ctx, 1)
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteRun_CancelledRun(t *testing.T) {
	svc, q, h := newTestRunService(t)
	ctx := context.Background()

	const repoID int64 = 10
	run := store.GetRunByIDRow{
		ID:           5,
		RepositoryID: repoID,
		Status:       store.FeatureRunStatusCancelled,
	}

	q.EXPECT().GetRunByID(ctx, run.ID).Return(run, nil)
	q.EXPECT().DeleteCancelledRun(ctx, run.ID).Return(nil)
	h.EXPECT().Publish(repoID, hub.EventRunUpdated)

	err := svc.DeleteRun(ctx, run.ID)
	require.NoError(t, err)
}

func TestDeleteRun_NotDeletable(t *testing.T) {
	svc, q, _ := newTestRunService(t)
	ctx := context.Background()

	run := pendingRun(10)
	q.EXPECT().GetRunByID(ctx, run.ID).Return(run, nil)

	err := svc.DeleteRun(ctx, run.ID)
	assert.ErrorIs(t, err, ErrRunNotDeletable)
}
