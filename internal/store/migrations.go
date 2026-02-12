package store

const createTableSQL = `
CREATE TABLE IF NOT EXISTS executions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    owner           TEXT NOT NULL,
    repo            TEXT NOT NULL,
    issue_number    INTEGER NOT NULL,
    installation_id INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    branch_name     TEXT NOT NULL DEFAULT '',
    pr_url          TEXT NOT NULL DEFAULT '',
    error_message   TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner, repo, issue_number)
);
`

func (s *Store) migrate() error {
	_, err := s.db.Exec(createTableSQL)
	return err
}
