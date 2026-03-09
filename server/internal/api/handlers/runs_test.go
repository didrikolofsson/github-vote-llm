package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRunsRouter(st store.Store) *gin.Engine {
	r := gin.New()
	h := handlers.NewRunsHandler(services.NewRunsService(st))
	r.GET("/runs", h.List)
	r.POST("/runs", h.Create)
	r.GET("/runs/:id", h.Get)
	r.POST("/runs/:id/retry", h.Retry)
	r.POST("/runs/:id/cancel", h.Cancel)
	return r
}

func stubExecution() *store.ExecutionModel {
	branch := "vote-llm/issue-1-test"
	return &store.ExecutionModel{
		ID:          1,
		Owner:       "owner",
		Repo:        "repo",
		IssueNumber: 1,
		Status:      "pending",
		Branch:      &branch,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// TestListRuns_Returns200WithRuns verifies the happy path returns runs.
func TestListRuns_Returns200WithRuns(t *testing.T) {
	exec := stubExecution()
	router := newRunsRouter(&store.MockStore{
		ListExecutionsFn: func(ctx context.Context, limit, offset int32) ([]*store.ExecutionModel, error) {
			return []*store.ExecutionModel{exec}, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []handlers.RunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != exec.ID {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// TestListRuns_InvalidLimit_Returns400 verifies that an out-of-range limit is rejected.
func TestListRuns_InvalidLimit_Returns400(t *testing.T) {
	router := newRunsRouter(&store.MockStore{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs?limit=999", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestListRuns_NegativeOffset_Returns400 verifies that a negative offset is rejected.
func TestListRuns_NegativeOffset_Returns400(t *testing.T) {
	router := newRunsRouter(&store.MockStore{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs?offset=-1", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateRun_Returns201WithRun verifies that a valid request creates and returns a run.
func TestCreateRun_Returns201WithRun(t *testing.T) {
	exec := stubExecution()
	router := newRunsRouter(&store.MockStore{
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return exec, nil
		},
	})

	body, _ := json.Marshal(map[string]any{"owner": "owner", "repo": "repo", "issue_number": 1})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp handlers.RunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.ID != exec.ID {
		t.Errorf("expected ID %d, got %d", exec.ID, resp.ID)
	}
}

// TestCreateRun_MissingFields_Returns400 verifies that required fields are validated.
func TestCreateRun_MissingFields_Returns400(t *testing.T) {
	router := newRunsRouter(&store.MockStore{})

	body, _ := json.Marshal(map[string]any{"owner": "owner"}) // missing repo and issue_number
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateRun_AlreadyExists_Returns409 verifies that a duplicate run returns 409.
func TestCreateRun_AlreadyExists_Returns409(t *testing.T) {
	router := newRunsRouter(&store.MockStore{
		CreateExecutionFn: func(ctx context.Context, owner, repo string, issueNumber int) (*store.ExecutionModel, error) {
			return nil, store.ErrAlreadyExists
		},
	})

	body, _ := json.Marshal(map[string]any{"owner": "owner", "repo": "repo", "issue_number": 1})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetRun_Returns200WithRun verifies that an existing run is returned.
func TestGetRun_Returns200WithRun(t *testing.T) {
	exec := stubExecution()
	router := newRunsRouter(&store.MockStore{
		GetExecutionByIDFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return exec, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs/1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp handlers.RunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.ID != exec.ID {
		t.Errorf("expected ID %d, got %d", exec.ID, resp.ID)
	}
}

// TestGetRun_NotFound_Returns404 verifies that a missing run returns 404.
func TestGetRun_NotFound_Returns404(t *testing.T) {
	router := newRunsRouter(&store.MockStore{
		GetExecutionByIDFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return nil, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs/999", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetRun_InvalidID_Returns400 verifies that a non-numeric ID returns 400.
func TestGetRun_InvalidID_Returns400(t *testing.T) {
	router := newRunsRouter(&store.MockStore{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs/notanid", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRetryRun_Returns200WithRun verifies that a retryable run is reset and returned.
func TestRetryRun_Returns200WithRun(t *testing.T) {
	exec := stubExecution()
	router := newRunsRouter(&store.MockStore{
		RetryExecutionFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return exec, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runs/1/retry", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRetryRun_NotRetryable_Returns409 verifies that retrying a non-retryable run returns 409.
func TestRetryRun_NotRetryable_Returns409(t *testing.T) {
	router := newRunsRouter(&store.MockStore{
		RetryExecutionFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return nil, pgx.ErrNoRows
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runs/1/retry", nil))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCancelRun_Returns200WithRun verifies that a cancellable run is cancelled and returned.
func TestCancelRun_Returns200WithRun(t *testing.T) {
	exec := stubExecution()
	router := newRunsRouter(&store.MockStore{
		CancelExecutionFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return exec, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runs/1/cancel", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCancelRun_NotCancellable_Returns409 verifies that cancelling a non-cancellable run returns 409.
func TestCancelRun_NotCancellable_Returns409(t *testing.T) {
	router := newRunsRouter(&store.MockStore{
		CancelExecutionFn: func(ctx context.Context, id int64) (*store.ExecutionModel, error) {
			return nil, pgx.ErrNoRows
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runs/1/cancel", nil))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestListRuns_StoreError_Returns500 verifies that a store error propagates as 500.
func TestListRuns_StoreError_Returns500(t *testing.T) {
	router := newRunsRouter(&store.MockStore{
		ListExecutionsFn: func(ctx context.Context, limit, offset int32) ([]*store.ExecutionModel, error) {
			return nil, errors.New("db error")
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
