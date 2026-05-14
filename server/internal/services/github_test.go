package services

import (
	"context"
	"errors"
	"testing"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestGithubService(t *testing.T) (*GithubService, *store.MockQuerier, *hub.MockHub) {
	t.Helper()
	ctrl := gomock.NewController(t)
	q := store.NewMockQuerier(ctrl)
	h := hub.NewMockHub(ctrl)
	svc := NewGithubService(GithubServiceDeps{
		DB:      nil,
		Queries: q,
		Env:     &config.Environment{JWT_SECRET: "test-secret"},
		Hub:     h,
		// AppClient left nil — not needed for webhook tests
	})
	return svc, q, h
}

func existingInstallation(orgID, installationID int64) store.GithubInstallation {
	return store.GithubInstallation{
		ID:                   1,
		OrganizationID:       orgID,
		GithubInstallationID: installationID,
		GithubAccountLogin:   "acme",
		GithubAccountType:    "Organization",
		SuspendedAt:          pgtype.Timestamptz{Valid: false},
	}
}

func TestHandleInstallationWebhook_Deleted(t *testing.T) {
	svc, q, h := newTestGithubService(t)
	ctx := context.Background()

	const (
		orgID          int64 = 10
		installationID int64 = 999
	)

	q.EXPECT().
		GetInstallationByInstallationID(ctx, installationID).
		Return(existingInstallation(orgID, installationID), nil)

	q.EXPECT().
		DeleteInstallationByInstallationID(ctx, installationID).
		Return(nil)

	h.EXPECT().PublishOrg(orgID, hub.EventInstallationRemoved)

	payload := InstallationWebhookPayload{Action: "deleted"}
	payload.Installation.ID = installationID

	err := svc.HandleInstallationWebhook(ctx, payload)
	require.NoError(t, err)
}

func TestHandleInstallationWebhook_Suspend(t *testing.T) {
	svc, q, h := newTestGithubService(t)
	ctx := context.Background()

	const (
		orgID          int64 = 10
		installationID int64 = 999
	)

	q.EXPECT().
		GetInstallationByInstallationID(ctx, installationID).
		Return(existingInstallation(orgID, installationID), nil)

	q.EXPECT().
		SetInstallationSuspendedByInstallationID(ctx, gomock.Any()).
		Return(nil)

	h.EXPECT().PublishOrg(orgID, hub.EventInstallationSuspended)

	payload := InstallationWebhookPayload{Action: "suspend"}
	payload.Installation.ID = installationID

	err := svc.HandleInstallationWebhook(ctx, payload)
	require.NoError(t, err)
}

func TestHandleInstallationWebhook_Unsuspend(t *testing.T) {
	svc, q, h := newTestGithubService(t)
	ctx := context.Background()

	const (
		orgID          int64 = 10
		installationID int64 = 999
	)

	q.EXPECT().
		GetInstallationByInstallationID(ctx, installationID).
		Return(existingInstallation(orgID, installationID), nil)

	q.EXPECT().
		SetInstallationSuspendedByInstallationID(ctx, store.SetInstallationSuspendedByInstallationIDParams{
			GithubInstallationID: installationID,
			SuspendedAt:          pgtype.Timestamptz{Valid: false},
		}).
		Return(nil)

	h.EXPECT().PublishOrg(orgID, hub.EventInstallationActive)

	payload := InstallationWebhookPayload{Action: "unsuspend"}
	payload.Installation.ID = installationID

	err := svc.HandleInstallationWebhook(ctx, payload)
	require.NoError(t, err)
}

func TestHandleInstallationWebhook_UnknownAction(t *testing.T) {
	svc, q, _ := newTestGithubService(t)
	ctx := context.Background()

	const installationID int64 = 999

	q.EXPECT().
		GetInstallationByInstallationID(ctx, installationID).
		Return(existingInstallation(10, installationID), nil)

	// No hub publish, no DB mutation expected for unknown actions.

	payload := InstallationWebhookPayload{Action: "new_permissions_accepted"}
	payload.Installation.ID = installationID

	err := svc.HandleInstallationWebhook(ctx, payload)
	require.NoError(t, err)
}

func TestHandleInstallationWebhook_NotFound(t *testing.T) {
	svc, q, _ := newTestGithubService(t)
	ctx := context.Background()

	const installationID int64 = 999

	// Installation not in our DB → silently ignore.
	q.EXPECT().
		GetInstallationByInstallationID(ctx, installationID).
		Return(store.GithubInstallation{}, pgx.ErrNoRows)

	payload := InstallationWebhookPayload{Action: "deleted"}
	payload.Installation.ID = installationID

	err := svc.HandleInstallationWebhook(ctx, payload)
	require.NoError(t, err)
}

func TestHandleInstallationWebhook_DBError(t *testing.T) {
	svc, q, _ := newTestGithubService(t)
	ctx := context.Background()

	dbErr := errors.New("db down")
	q.EXPECT().
		GetInstallationByInstallationID(ctx, gomock.Any()).
		Return(store.GithubInstallation{}, dbErr)

	payload := InstallationWebhookPayload{Action: "deleted"}
	payload.Installation.ID = 1

	err := svc.HandleInstallationWebhook(ctx, payload)
	assert.ErrorIs(t, err, dbErr)
}
