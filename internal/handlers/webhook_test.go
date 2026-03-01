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

func TestHandleIssueEvent_AlreadyExists_NotRetryable_Returns200(t *testing.T) {
	mockStore := &store.MockStore{
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.Execution, error) {
			return nil, errors.New("execution already exists for this issue")
		},
		// Execution exists but is not in failed state (e.g. in_progress or succeeded).
		ResetFailedExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.Execution, error) {
			return nil, nil
		},
	}

	// WebhookHandler requires a ClientFactory — pass nil; the handler returns
	// before reaching the factory when the execution is not retryable.
	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)

	router := gin.New()
	router.POST("/webhook", handler.HandleGithubWebhook)

	payload := buildLabeledIssuePayload(t, 123)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleIssueEvent_AlreadyExists_ResetDBError_Returns500(t *testing.T) {
	mockStore := &store.MockStore{
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.Execution, error) {
			return nil, errors.New("execution already exists for this issue")
		},
		ResetFailedExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.Execution, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)

	router := gin.New()
	router.POST("/webhook", handler.HandleGithubWebhook)

	payload := buildLabeledIssuePayload(t, 123)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleIssueEvent_DBError_Returns500(t *testing.T) {
	mockStore := &store.MockStore{
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.Execution, error) {
			return nil, errDBFailure
		},
	}

	handler := handlers.NewWebhookHandler(nil, logger.New(), "/tmp", mockStore)

	router := gin.New()
	router.POST("/webhook", handler.HandleGithubWebhook)

	payload := buildLabeledIssuePayload(t, 123)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
