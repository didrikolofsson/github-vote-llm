CREATE TABLE github_install_states (
    nonce TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE github_installation_repositories (
    installation_id BIGINT NOT NULL REFERENCES github_installations (id) ON DELETE CASCADE,
    github_repository_id BIGINT NOT NULL,
    repository_name TEXT NOT NULL,
    repository_full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (
        installation_id,
        github_repository_id
    )
);

ALTER TABLE organization_repositories RENAME TO repositories;

ALTER TABLE user_authorization_codes RENAME TO authorization_codes;

ALTER TABLE user_refresh_tokens RENAME TO refresh_tokens;