package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath, logger.New())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCanProcess_NoRecord(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	ok, err := s.CanProcess(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("CanProcess: %v", err)
	}
	if !ok {
		t.Error("expected CanProcess=true for missing record")
	}
}

func TestCanProcess_Pending(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.CreateExecution(ctx, &ExecutionRecord{
		Owner: "owner", Repo: "repo", IssueNumber: 1, Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	ok, err := s.CanProcess(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("CanProcess: %v", err)
	}
	if ok {
		t.Error("expected CanProcess=false for pending record")
	}
}

func TestCanProcess_Failed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.CreateExecution(ctx, &ExecutionRecord{
		Owner: "owner", Repo: "repo", IssueNumber: 1, Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	if err := s.UpdateStatus(ctx, id, "failed", "", "something broke"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	ok, err := s.CanProcess(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("CanProcess: %v", err)
	}
	if !ok {
		t.Error("expected CanProcess=true for failed record (allows retry)")
	}
}

func TestCanProcess_Completed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.CreateExecution(ctx, &ExecutionRecord{
		Owner: "owner", Repo: "repo", IssueNumber: 1, Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	if err := s.UpdateStatus(ctx, id, "completed", "https://github.com/pr/1", ""); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	ok, err := s.CanProcess(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("CanProcess: %v", err)
	}
	if ok {
		t.Error("expected CanProcess=false for completed record")
	}
}

func TestGetExecution(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.CreateExecution(ctx, &ExecutionRecord{
		Owner: "owner", Repo: "repo", IssueNumber: 42, Status: "running", BranchName: "vote-llm/issue-42-foo",
	})
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	rec, err := s.GetExecution(ctx, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if rec == nil {
		t.Fatal("expected non-nil record")
	}
	if rec.ID != id {
		t.Errorf("ID = %d, want %d", rec.ID, id)
	}
	if rec.Status != "running" {
		t.Errorf("Status = %q, want %q", rec.Status, "running")
	}
	if rec.BranchName != "vote-llm/issue-42-foo" {
		t.Errorf("BranchName = %q, want %q", rec.BranchName, "vote-llm/issue-42-foo")
	}
}

func TestGetExecution_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	rec, err := s.GetExecution(ctx, "owner", "repo", 999)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if rec != nil {
		t.Error("expected nil record for missing issue")
	}
}

func TestResetExecution(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.CreateExecution(ctx, &ExecutionRecord{
		Owner: "owner", Repo: "repo", IssueNumber: 1, Status: "completed",
	})
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	if err := s.ResetExecution(ctx, "owner", "repo", 1); err != nil {
		t.Fatalf("ResetExecution: %v", err)
	}

	ok, err := s.CanProcess(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("CanProcess: %v", err)
	}
	if !ok {
		t.Error("expected CanProcess=true after reset")
	}
}
