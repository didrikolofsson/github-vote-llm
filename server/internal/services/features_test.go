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

func newTestFeaturesService(t *testing.T) (*FeaturesService, *store.MockQuerier, *hub.MockHub) {
	t.Helper()
	ctrl := gomock.NewController(t)
	q := store.NewMockQuerier(ctrl)
	h := hub.NewMockHub(ctrl)
	svc := NewFeaturesService(nil, q, h)
	return svc, q, h
}

func TestToggleVote_AddVote(t *testing.T) {
	svc, q, h := newTestFeaturesService(t)
	ctx := context.Background()

	const featureID int64 = 1
	const voterToken = "tok-abc"

	// No existing vote → add
	q.EXPECT().
		GetFeatureVote(ctx, store.GetFeatureVoteParams{FeatureID: featureID, VoterToken: voterToken}).
		Return(store.FeatureVote{}, pgx.ErrNoRows)

	q.EXPECT().
		AddFeatureVote(ctx, store.AddFeatureVoteParams{
			FeatureID:  featureID,
			VoterToken: voterToken,
			Reason:     "great idea",
			Urgency:    store.NullVoteUrgencyType{},
		}).
		Return(store.FeatureVote{ID: 10, FeatureID: featureID}, nil)

	q.EXPECT().
		CountFeatureVotes(ctx, featureID).
		Return(int64(3), nil)

	h.EXPECT().Publish(featureID, hub.EventFeatureUpdated)

	count, err := svc.ToggleVote(ctx, featureID, voterToken, "great idea", store.NullVoteUrgencyType{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestToggleVote_RemoveVote(t *testing.T) {
	svc, q, h := newTestFeaturesService(t)
	ctx := context.Background()

	const featureID int64 = 1
	const voterToken = "tok-abc"

	// Existing vote → remove
	q.EXPECT().
		GetFeatureVote(ctx, store.GetFeatureVoteParams{FeatureID: featureID, VoterToken: voterToken}).
		Return(store.FeatureVote{ID: 10, FeatureID: featureID}, nil)

	q.EXPECT().
		RemoveFeatureVote(ctx, store.RemoveFeatureVoteParams{FeatureID: featureID, VoterToken: voterToken}).
		Return(nil)

	q.EXPECT().
		CountFeatureVotes(ctx, featureID).
		Return(int64(2), nil)

	h.EXPECT().Publish(featureID, hub.EventFeatureUpdated)

	count, err := svc.ToggleVote(ctx, featureID, voterToken, "", store.NullVoteUrgencyType{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestToggleVote_DBError(t *testing.T) {
	svc, q, _ := newTestFeaturesService(t)
	ctx := context.Background()

	dbErr := errors.New("connection refused")
	q.EXPECT().
		GetFeatureVote(ctx, gomock.Any()).
		Return(store.FeatureVote{}, dbErr)

	_, err := svc.ToggleVote(ctx, 1, "tok", "reason", store.NullVoteUrgencyType{})
	assert.ErrorIs(t, err, dbErr)
}

func TestToggleVote_CountError(t *testing.T) {
	svc, q, _ := newTestFeaturesService(t)
	ctx := context.Background()

	const featureID int64 = 1

	q.EXPECT().
		GetFeatureVote(ctx, gomock.Any()).
		Return(store.FeatureVote{}, pgx.ErrNoRows)

	q.EXPECT().
		AddFeatureVote(ctx, gomock.Any()).
		Return(store.FeatureVote{}, nil)

	countErr := errors.New("count failed")
	q.EXPECT().
		CountFeatureVotes(ctx, featureID).
		Return(int64(0), countErr)

	_, err := svc.ToggleVote(ctx, featureID, "tok", "reason", store.NullVoteUrgencyType{})
	assert.ErrorIs(t, err, countErr)
}
