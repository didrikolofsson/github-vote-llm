DROP TABLE IF EXISTS github_connections;

CREATE TABLE github_installations (
    id                     BIGSERIAL PRIMARY KEY,
    organization_id        BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    github_installation_id BIGINT NOT NULL UNIQUE,
    github_account_login   TEXT NOT NULL,
    github_account_id      BIGINT NOT NULL,
    github_account_type    TEXT NOT NULL,
    repository_selection   TEXT NOT NULL,
    suspended_at           TIMESTAMPTZ,
    installed_by_user_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE github_installation_repositories (
    installation_id        BIGINT NOT NULL REFERENCES github_installations(id) ON DELETE CASCADE,
    github_repository_id   BIGINT NOT NULL,
    repository_name        TEXT NOT NULL,
    repository_full_name   TEXT NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (installation_id, github_repository_id)
);

CREATE TABLE github_install_states (
    nonce       TEXT PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX github_install_states_user_id_idx ON github_install_states (user_id);
