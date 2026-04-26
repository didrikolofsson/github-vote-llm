DROP TABLE IF EXISTS github_install_states;

DROP TABLE IF EXISTS github_installation_repositories;

ALTER TABLE repositories RENAME TO organization_repositories;

ALTER TABLE authorization_codes RENAME TO user_authorization_codes;

ALTER TABLE refresh_tokens RENAME TO user_refresh_tokens;