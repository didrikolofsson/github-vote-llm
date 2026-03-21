-- name: CreateOrganization :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING id,
    name,
    created_at,
    updated_at;
-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;
-- name: GetOrganizationByID :one
SELECT id,
    name,
    created_at,
    updated_at
FROM organizations
WHERE id = $1;
-- name: UpdateOrganizationByID :one
UPDATE organizations
SET name = $2
WHERE id = $1
RETURNING id,
    name,
    created_at,
    updated_at;
-- name: DeleteOrganizationByID :exec
DELETE FROM organizations
WHERE id = $1;