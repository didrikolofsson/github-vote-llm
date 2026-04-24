DROP TABLE IF EXISTS github_install_states;
DROP TABLE IF EXISTS github_installation_repositories;
DROP TABLE IF EXISTS github_installations;

CREATE TABLE github_connections (
    user_id                BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE PRIMARY KEY,
    access_token_encrypted TEXT NOT NULL,
    refresh_token          TEXT,
    token_expires_at       TIMESTAMPTZ,
    github_user_id         BIGINT,
    github_login           TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);
