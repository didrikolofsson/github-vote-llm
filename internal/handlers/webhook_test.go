package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/didrikolofsson/github-vote-llm/internal/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// buildLabeledIssuePayload returns a minimal IssuesEvent JSON payload for a
// "labeled" action on an issue that has both feature-request and approved-for-dev labels.
func buildLabeledIssuePayload(t *testing.T, installationID int64) []byte {
	t.Helper()
	featureLabel := "feature-request"
	approvedLabel := "approved-for-dev"
	action := "labeled"
	issueNum := 1
	title := "test issue"
	owner := "owner"
	repoName := "repo"
	installID := installationID

	event := gh.IssuesEvent{
		Action: &action,
		Label:  &gh.Label{Name: &approvedLabel},
		Issue: &gh.Issue{
			Number: &issueNum,
			Title:  &title,
			Labels: []*gh.Label{
				{Name: &featureLabel},
				{Name: &approvedLabel},
			},
		},
		Repo: &gh.Repository{
			Name: &repoName,
			Owner: &gh.User{
				Login: &owner,
			},
		},
		Installation: &gh.Installation{
			ID: &installID,
		},
	}
	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	return b
}

var errDBFailure = errors.New("db connection lost")

// noExecution is a GetExecutionByOwnerRepoIssueNumber stub that returns nil (no
// existing record), causing the handler to proceed to CreateExecution.
func noExecution(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
	return nil, nil
}

func postWebhook(t *testing.T, handler *handlers.WebhookHandler, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/webhook", handler.HandleGithubWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestHandleIssueEvent_AlreadySucceeded_Returns200 verifies that an issue whose
// execution already completed successfully is silently skipped (200, no agent run).
func TestHandleIssueEvent_AlreadySucceeded_Returns200(t *testing.T) {
	status := "success"
	mockStore := &store.MockStore{
		GetExecutionByOwnerRepoIssueNumberFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return &store.ExecutionModel{Status: status}, nil
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postWebhook(t, handler, buildLabeledIssuePayload(t, 123))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleIssueEvent_FailedExecution_ResetDBError_Returns500 verifies that a
// DB error when resetting a previously failed execution results in a 500.
func TestHandleIssueEvent_FailedExecution_ResetDBError_Returns500(t *testing.T) {
	status := "failed"
	mockStore := &store.MockStore{
		GetExecutionByOwnerRepoIssueNumberFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return &store.ExecutionModel{Status: status}, nil
		},
		ResetFailedExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postWebhook(t, handler, buildLabeledIssuePayload(t, 123))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleIssueEvent_GetExecutionDBError_Returns500 verifies that a DB error
// when looking up an existing execution results in a 500.
func TestHandleIssueEvent_GetExecutionDBError_Returns500(t *testing.T) {
	mockStore := &store.MockStore{
		GetExecutionByOwnerRepoIssueNumberFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postWebhook(t, handler, buildLabeledIssuePayload(t, 123))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleIssueEvent_CreateExecutionDBError_Returns500 verifies that a DB
// error when creating a new execution record results in a 500.
func TestHandleIssueEvent_CreateExecutionDBError_Returns500(t *testing.T) {
	mockStore := &store.MockStore{
		GetExecutionByOwnerRepoIssueNumberFn: noExecution,
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postWebhook(t, handler, buildLabeledIssuePayload(t, 123))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// postReactionWebhook is a helper that POSTs a reaction webhook payload.
func postReactionWebhook(t *testing.T, handler *handlers.WebhookHandler, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/webhook", handler.HandleGithubWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "reaction")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// buildReactionPayload builds a minimal reaction event JSON payload.
func buildReactionPayload(t *testing.T, action, content, subjectType string, hasFeatureRequest bool, installationID int64) []byte {
	t.Helper()

	featureLabel := "feature-request"
	labels := []map[string]string{}
	if hasFeatureRequest {
		labels = append(labels, map[string]string{"name": featureLabel})
	}

	payload := map[string]any{
		"action":       action,
		"subject_type": subjectType,
		"reaction":     map[string]string{"content": content},
		"issue": map[string]any{
			"number": 42,
			"title":  "some feature",
			"labels": labels,
		},
		"repository": map[string]any{
			"name": "repo",
			"owner": map[string]string{
				"login": "owner",
			},
		},
		"installation": map[string]any{
			"id": installationID,
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal reaction payload: %v", err)
	}
	return b
}

// TestHandleReactionEvent_NonCreated_Returns200 verifies that reactions with
// action != "created" are silently skipped.
func TestHandleReactionEvent_NonCreated_Returns200(t *testing.T) {
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", &store.MockStore{})
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "deleted", "+1", "issue", true, 0))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleReactionEvent_NonThumbsUp_Returns200 verifies that non-+1
// reactions are silently skipped.
func TestHandleReactionEvent_NonThumbsUp_Returns200(t *testing.T) {
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", &store.MockStore{})
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "created", "heart", "issue", true, 0))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleReactionEvent_NonIssueSubject_Returns200 verifies that reactions
// on non-issue subjects (e.g., comments) are silently skipped.
func TestHandleReactionEvent_NonIssueSubject_Returns200(t *testing.T) {
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", &store.MockStore{})
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "created", "+1", "pull_request", true, 0))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleReactionEvent_NoFeatureRequest_Returns200 verifies that a +1 on
// an issue without the feature-request label is silently skipped.
func TestHandleReactionEvent_NoFeatureRequest_Returns200(t *testing.T) {
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", &store.MockStore{})
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "created", "+1", "issue", false, 0))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleReactionEvent_VoteDBError_Returns500 verifies that a DB error
// when recording a vote results in a 500.
func TestHandleReactionEvent_VoteDBError_Returns500(t *testing.T) {
	mockStore := &store.MockStore{
		IncrementIssueVoteFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.IssueVoteModel, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "created", "+1", "issue", true, 123))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleReactionEvent_BelowThreshold_Returns200 verifies that a +1 below
// the vote threshold records the vote but does not attempt to add a label.
func TestHandleReactionEvent_BelowThreshold_Returns200(t *testing.T) {
	mockStore := &store.MockStore{
		GetRepoConfigFn: func(ctx context.Context, owner, repo string) (*store.RepoConfigModel, error) {
			return &store.RepoConfigModel{
				LabelCandidate: "candidate",
				VoteThreshold:  10,
			}, nil
		},
		IncrementIssueVoteFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.IssueVoteModel, error) {
			return &store.IssueVoteModel{VoteCount: 3}, nil
		},
	}

	// nil factory is safe here: threshold not reached, so no GitHub API call.
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)
	w := postReactionWebhook(t, handler, buildReactionPayload(t, "created", "+1", "issue", true, 123))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
