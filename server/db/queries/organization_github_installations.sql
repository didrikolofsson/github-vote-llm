-- name: UpsertInstallation :one
INSERT INTO
    github_installations (
        organization_id,
        github_installation_id,
        github_account_login,
        github_account_id,
        github_account_type,
        repository_selection,
        suspended_at,
        installed_by_user_id
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8
    )
ON CONFLICT (organization_id) DO
UPDATE
SET
    github_installation_id = EXCLUDED.github_installation_id,
    github_account_login = EXCLUDED.github_account_login,
    github_account_id = EXCLUDED.github_account_id,
    github_account_type = EXCLUDED.github_account_type,
    repository_selection = EXCLUDED.repository_selection,
    suspended_at = EXCLUDED.suspended_at,
    installed_by_user_id = EXCLUDED.installed_by_user_id,
    updated_at = now()
RETURNING
    *;
-- name: GetInstallationByOrgID :one
SELECT gi.*, u.username AS installed_by_user_name
FROM
    github_installations gi
    INNER JOIN users u ON gi.installed_by_user_id = u.id
WHERE
    gi.organization_id = $1;
-- name: GetInstallationByInstallationID :one
SELECT * FROM github_installations WHERE github_installation_id = $1;
-- name: SetInstallationSuspendedByInstallationID :exec
UPDATE github_installations
SET
    suspended_at = $2,
    updated_at = now()
WHERE
    github_installation_id = $1;
-- name: DeleteInstallationByInstallationID :exec
DELETE FROM github_installations WHERE github_installation_id = $1;
-- name: DeleteInstallationByOrgID :exec
DELETE FROM github_installations WHERE organization_id = $1;
-- name: DeleteInstallationByID :exec
DELETE FROM github_installations WHERE id = $1;