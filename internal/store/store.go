package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	_ "modernc.org/sqlite"
)

// ExecutionRecord represents a row in the executions table.
type ExecutionRecord struct {
	ID             int64
	Owner          string
	Repo           string
	IssueNumber    int
	InstallationID int64
	Status         string
	BranchName     string
	PRURL          string
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Store provides SQLite-backed persistence for execution records.
type Store struct {
	db  *sql.DB
	log *logger.Logger
}

// NewStore opens (or creates) the SQLite database and runs migrations.
func NewStore(dbPath string, log *logger.Logger) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	s := &Store{db: db, log: log.Named("store")}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// CanProcess returns true if the issue can be processed (no record or status is "failed").
func (s *Store) CanProcess(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	var status string
	err := s.db.QueryRowContext(ctx,
		"SELECT status FROM executions WHERE owner = ? AND repo = ? AND issue_number = ?",
		owner, repo, issueNumber,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("query execution: %w", err)
	}
	return status == "failed", nil
}

// CreateExecution inserts or replaces an execution record.
func (s *Store) CreateExecution(ctx context.Context, rec *ExecutionRecord) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO executions (owner, repo, issue_number, installation_id, status, branch_name, pr_url, error_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		rec.Owner, rec.Repo, rec.IssueNumber, rec.InstallationID, rec.Status, rec.BranchName, rec.PRURL, rec.ErrorMessage,
	)
	if err != nil {
		return 0, fmt.Errorf("create execution: %w", err)
	}
	return result.LastInsertId()
}

// UpdateStatus updates the status, PR URL, and error message of an execution.
func (s *Store) UpdateStatus(ctx context.Context, id int64, status, prURL, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE executions SET status = ?, pr_url = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, prURL, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update execution status: %w", err)
	}
	return nil
}

// GetExecution returns the execution record for a given issue, or nil if not found.
func (s *Store) GetExecution(ctx context.Context, owner, repo string, issueNumber int) (*ExecutionRecord, error) {
	rec := &ExecutionRecord{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, owner, repo, issue_number, installation_id, status, branch_name, pr_url, error_message, created_at, updated_at
		 FROM executions WHERE owner = ? AND repo = ? AND issue_number = ?`,
		owner, repo, issueNumber,
	).Scan(&rec.ID, &rec.Owner, &rec.Repo, &rec.IssueNumber, &rec.InstallationID, &rec.Status, &rec.BranchName, &rec.PRURL, &rec.ErrorMessage, &rec.CreatedAt, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	return rec, nil
}

// ResetExecution deletes the execution record for a given issue (for manual retry).
func (s *Store) ResetExecution(ctx context.Context, owner, repo string, issueNumber int) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM executions WHERE owner = ? AND repo = ? AND issue_number = ?",
		owner, repo, issueNumber,
	)
	if err != nil {
		return fmt.Errorf("reset execution: %w", err)
	}
	return nil
}
