CREATE TABLE executions (
    id            BIGSERIAL PRIMARY KEY,
    owner         TEXT NOT NULL,
    repo          TEXT NOT NULL,
    issue_number  INT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    branch        TEXT,
    pr_url        TEXT,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, repo, issue_number)
);
