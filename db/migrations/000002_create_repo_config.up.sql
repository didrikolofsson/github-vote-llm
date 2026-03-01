CREATE TABLE repo_config (
    id                BIGSERIAL PRIMARY KEY,
    owner             TEXT NOT NULL,
    repo              TEXT NOT NULL,
    label_approved    TEXT,
    label_in_progress TEXT,
    label_done        TEXT,
    label_failed      TEXT,
    vote_threshold    INT,
    timeout_minutes   INT,
    max_budget_usd    NUMERIC(10,2),
    anthropic_api_key TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, repo)
);
