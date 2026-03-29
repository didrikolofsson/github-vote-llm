-- name: CreateOrganization :one
INSERT INTO organizations (name, slug)
VALUES ($1, $2)
RETURNING id,
    name,
    slug,
    created_at,
    updated_at;
-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;
-- name: GetOrganizationByID :one
SELECT id,
    name,
    slug,
    created_at,
    updated_at
FROM organizations
WHERE id = $1;
-- name: GetOrganizationBySlug :one
SELECT id,
    name,
    slug,
    created_at,
    updated_at
FROM organizations
WHERE slug = $1;
-- name: UpdateOrganizationByID :one
UPDATE organizations
SET name = $2
WHERE id = $1
RETURNING id,
    name,
    slug,
    created_at,
    updated_at;
-- name: UpdateOrganizationSlug :one
UPDATE organizations
SET slug = $2,
    updated_at = now()
WHERE id = $1
RETURNING id,
    name,
    slug,
    created_at,
    updated_at;
-- name: DeleteOrganizationByID :exec
DELETE FROM organizations
WHERE id = $1;
-- name: ListOrganizationsForUser :many
SELECT o.id,
    o.name,
    o.slug,
    o.created_at,
    o.updated_at
FROM organizations o
JOIN organization_members om ON om.organization_id = o.id
WHERE om.user_id = $1
ORDER BY o.name;
