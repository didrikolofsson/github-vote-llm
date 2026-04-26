-- name: UpsertInstallation :one
INSERT INTO github_installations (
        organization_id,
        github_installation_id,
        github_account_login,
        github_account_id,
        github_account_type,
        repository_selection,
        suspended_at,
        installed_by_user_id,
        state
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (organization_id) DO
UPDATE
SET github_installation_id = EXCLUDED.github_installation_id,
    github_account_login = EXCLUDED.github_account_login,
    github_account_id = EXCLUDED.github_account_id,
    github_account_type = EXCLUDED.github_account_type,
    repository_selection = EXCLUDED.repository_selection,
    suspended_at = EXCLUDED.suspended_at,
    installed_by_user_id = EXCLUDED.installed_by_user_id,
    state = EXCLUDED.state,
    updated_at = now()
RETURNING *;
-- name: GetInstallationByOrgID :one
SELECT *
FROM github_installations
WHERE organization_id = $1;
-- name: GetInstallationByGithubID :one
SELECT *
FROM github_installations
WHERE github_installation_id = $1;
-- name: SetInstallationSuspendedByGithubID :exec
UPDATE github_installations
SET suspended_at = $2,
    updated_at = now()
WHERE github_installation_id = $1;
-- name: DeleteInstallationByGithubID :exec
DELETE FROM github_installations
WHERE github_installation_id = $1;
-- name: DeleteInstallationByOrgID :exec
DELETE FROM github_installations
WHERE organization_id = $1;
-- name: AddInstallationRepository :exec
INSERT INTO github_installation_repositories (
        installation_id,
        github_repository_id,
        repository_name,
        repository_full_name
    )
VALUES ($1, $2, $3, $4) ON CONFLICT (installation_id, github_repository_id) DO
UPDATE
SET repository_name = EXCLUDED.repository_name,
    repository_full_name = EXCLUDED.repository_full_name;
-- name: RemoveInstallationRepository :exec
DELETE FROM github_installation_repositories
WHERE installation_id = $1
    AND github_repository_id = $2;
-- name: ListInstallationRepositories :many
SELECT *
FROM github_installation_repositories
WHERE installation_id = $1
ORDER BY repository_full_name;
-- name: ClearInstallationRepositories :exec
DELETE FROM github_installation_repositories
WHERE installation_id = $1;
-- name: DeleteInstallationByID :exec
DELETE FROM github_installations
WHERE id = $1;