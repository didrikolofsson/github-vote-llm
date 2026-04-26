-- name: GetRepository :one
SELECT * FROM organization_repositories WHERE id = $1;
-- name: GetRepositoryByOwnerAndName :one
SELECT *
FROM organization_repositories
WHERE
    organization_id = $1
    AND owner = $2
    AND name = $3;
-- name: GetPublicRepositoryByOrgAndName :one
SELECT r.*
FROM
    organization_repositories r
    JOIN organizations o ON o.id = r.organization_id
WHERE
    o.slug = $1
    AND r.name = $2
    AND r.portal_public = true;
-- name: ListRepositoriesForOrganization :many
SELECT *
FROM organization_repositories
WHERE
    organization_id = $1
ORDER BY created_at DESC;
-- name: AddRepository :one
INSERT INTO
    organization_repositories (organization_id, owner, name)
VALUES ($1, $2, $3)
RETURNING
    *;
-- name: SetRepositoryPortalPublic :one
UPDATE organization_repositories
SET
    portal_public = $2,
    updated_at = now()
WHERE
    id = $1
RETURNING
    *;
-- name: RemoveRepository :exec
DELETE FROM organization_repositories WHERE id = $1;
-- name: GetRepositoryFeatureCount :one
SELECT COUNT(*) AS count FROM features WHERE repository_id = $1;
-- name: GetRepositoryByFeatureID :one
SELECT r.*
FROM
    organization_repositories r
    JOIN features f ON f.repository_id = r.id
WHERE
    f.id = $1;