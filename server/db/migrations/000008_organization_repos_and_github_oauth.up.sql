-- Drop old repo_config (will be recreated with fresh schema)
DROP TABLE IF EXISTS repo_config CASCADE;

-- Junction: orgs can have many repos, same repo can be in multiple orgs
CREATE TABLE organization_repositories (
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, owner, repo)
);

CREATE INDEX organization_repositories_org_id_idx ON organization_repositories (organization_id);

-- GitHub OAuth tokens (per user)
CREATE TABLE github_connections (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE PRIMARY KEY,
    access_token_encrypted TEXT NOT NULL,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    github_user_id BIGINT,
    github_login TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Recreate repo_config (keyed by owner+repo, shared across orgs)
CREATE TABLE repo_config (
    id BIGSERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    label_approved TEXT,
    label_in_progress TEXT,
    label_done TEXT,
    label_failed TEXT,
    label_feature_request TEXT,
    vote_threshold INT,
    timeout_minutes INT,
    max_budget_usd NUMERIC(10, 2),
    anthropic_api_key TEXT,
    is_board_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner, repo)
);
