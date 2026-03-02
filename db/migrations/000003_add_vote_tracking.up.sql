ALTER TABLE repo_config ADD COLUMN label_candidate TEXT;

CREATE TABLE issue_votes (
    id           BIGSERIAL PRIMARY KEY,
    owner        TEXT NOT NULL,
    repo         TEXT NOT NULL,
    issue_number INT  NOT NULL,
    vote_count   INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, repo, issue_number)
);
