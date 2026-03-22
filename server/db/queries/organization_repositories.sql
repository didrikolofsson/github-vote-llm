-- name: ListRepositoriesForOrganization :many
SELECT organization_id, owner, repo, created_at
FROM organization_repositories
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: AddOrganizationRepository :one
INSERT INTO organization_repositories (organization_id, owner, repo)
VALUES ($1, $2, $3)
RETURNING organization_id, owner, repo, created_at;

-- name: RemoveOrganizationRepository :exec
DELETE FROM organization_repositories
WHERE organization_id = $1 AND owner = $2 AND repo = $3;

-- name: GetOrganizationRepository :one
SELECT organization_id, owner, repo, created_at
FROM organization_repositories
WHERE organization_id = $1 AND owner = $2 AND repo = $3;
