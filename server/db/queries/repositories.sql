-- name: GetRepository :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByOwnerAndName :one
SELECT * FROM repositories
WHERE organization_id = $1 AND owner = $2 AND name = $3;

-- name: GetPublicRepositoryByOrgAndName :one
SELECT r.* FROM repositories r
JOIN organizations o ON o.id = r.organization_id
WHERE o.slug = $1 AND r.name = $2 AND r.portal_public = true;

-- name: ListRepositoriesForOrganization :many
SELECT * FROM repositories
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: AddRepository :one
INSERT INTO repositories (organization_id, owner, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SetRepositoryPortalPublic :one
UPDATE repositories
SET portal_public = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RemoveRepository :exec
DELETE FROM repositories WHERE id = $1;
