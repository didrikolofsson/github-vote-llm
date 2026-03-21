-- name: AddOrganizationMember :one
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING organization_id,
    user_id,
    role;
-- name: RemoveOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = $1
    AND user_id = $2;
-- name: GetOrganizationMembers :many
SELECT organization_id,
    user_id,
    role
FROM organization_members
WHERE organization_id = $1;