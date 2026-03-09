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
)

func newReposRouter(st store.Store) *gin.Engine {
	r := gin.New()
	h := handlers.NewReposHandler(services.NewReposService(st))
	r.GET("/repos", h.List)
	r.GET("/repos/:owner/:repo/config", h.GetConfig)
	r.PUT("/repos/:owner/:repo/config", h.UpdateConfig)
	return r
}

func stubRepoConfig() *store.RepoConfigModel {
	return &store.RepoConfigModel{
		ID:                  1,
		Owner:               "owner",
		Repo:                "repo",
		LabelApproved:       "approved-for-dev",
		LabelInProgress:     "llm-in-progress",
		LabelDone:           "llm-pr-created",
		LabelFailed:         "llm-failed",
		LabelFeatureRequest: "feature-request",
		VoteThreshold:       3,
		TimeoutMinutes:      30,
		MaxBudgetUsd:        5.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

// TestListRepos_Returns200WithConfigs verifies the happy path returns repo configs.
func TestListRepos_Returns200WithConfigs(t *testing.T) {
	cfg := stubRepoConfig()
	router := newReposRouter(&store.MockStore{
		ListRepoConfigsFn: func(ctx context.Context) ([]*store.RepoConfigModel, error) {
			return []*store.RepoConfigModel{cfg}, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repos", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []handlers.RepoConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != cfg.ID {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// TestListRepos_Empty_Returns200WithEmptyArray verifies that an empty store returns an empty array (not null).
func TestListRepos_Empty_Returns200WithEmptyArray(t *testing.T) {
	router := newReposRouter(&store.MockStore{
		ListRepoConfigsFn: func(ctx context.Context) ([]*store.RepoConfigModel, error) {
			return []*store.RepoConfigModel{}, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repos", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body != "[]\n" && body != "[]" {
		t.Errorf("expected empty array, got %s", body)
	}
}

// TestListRepos_StoreError_Returns500 verifies that a store error propagates as 500.
func TestListRepos_StoreError_Returns500(t *testing.T) {
	router := newReposRouter(&store.MockStore{
		ListRepoConfigsFn: func(ctx context.Context) ([]*store.RepoConfigModel, error) {
			return nil, errors.New("db error")
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repos", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetRepoConfig_Returns200WithConfig verifies that an existing config is returned.
func TestGetRepoConfig_Returns200WithConfig(t *testing.T) {
	cfg := stubRepoConfig()
	router := newReposRouter(&store.MockStore{
		GetRepoConfigFn: func(ctx context.Context, owner, repo string) (*store.RepoConfigModel, error) {
			return cfg, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repos/owner/repo/config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp handlers.RepoConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Owner != "owner" || resp.Repo != "repo" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// TestGetRepoConfig_NotFound_Returns404 verifies that a missing config returns 404.
func TestGetRepoConfig_NotFound_Returns404(t *testing.T) {
	router := newReposRouter(&store.MockStore{
		GetRepoConfigFn: func(ctx context.Context, owner, repo string) (*store.RepoConfigModel, error) {
			return nil, nil
		},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repos/owner/repo/config", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestUpdateRepoConfig_Returns200WithUpdatedConfig verifies that a valid update returns the config.
func TestUpdateRepoConfig_Returns200WithUpdatedConfig(t *testing.T) {
	cfg := stubRepoConfig()
	cfg.VoteThreshold = 5
	router := newReposRouter(&store.MockStore{
		UpsertRepoConfigFn: func(ctx context.Context, params store.UpsertRepoConfigParams) (*store.RepoConfigModel, error) {
			return cfg, nil
		},
	})

	threshold := int32(5)
	body, _ := json.Marshal(map[string]any{"vote_threshold": threshold})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/repos/owner/repo/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp handlers.RepoConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.VoteThreshold != 5 {
		t.Errorf("expected vote_threshold 5, got %d", resp.VoteThreshold)
	}
}

// TestUpdateRepoConfig_InvalidJSON_Returns400 verifies that malformed JSON is rejected.
func TestUpdateRepoConfig_InvalidJSON_Returns400(t *testing.T) {
	router := newReposRouter(&store.MockStore{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/repos/owner/repo/config", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestUpdateRepoConfig_StoreError_Returns500 verifies that a store error propagates as 500.
func TestUpdateRepoConfig_StoreError_Returns500(t *testing.T) {
	router := newReposRouter(&store.MockStore{
		UpsertRepoConfigFn: func(ctx context.Context, params store.UpsertRepoConfigParams) (*store.RepoConfigModel, error) {
			return nil, errors.New("db error")
		},
	})

	body, _ := json.Marshal(map[string]any{"vote_threshold": 5})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/repos/owner/repo/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
