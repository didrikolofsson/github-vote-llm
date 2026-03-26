-- name: GetRepository :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByOwnerAndName :one
SELECT * FROM repositories
WHERE organization_id = $1 AND owner = $2 AND name = $3;

-- name: ListRepositoriesForOrganization :many
SELECT * FROM repositories
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: AddRepository :one
INSERT INTO repositories (organization_id, owner, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveRepository :exec
DELETE FROM repositories WHERE id = $1;
