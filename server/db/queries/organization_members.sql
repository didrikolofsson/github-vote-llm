-- name: GetOrganizationMembershipByUserID :one
SELECT organization_id,
    user_id,
    role
FROM organization_members
WHERE user_id = $1;

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

-- name: GetOrganizationMembersWithUser :many
SELECT om.organization_id, om.user_id, om.role, u.email
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1;

-- name: UpdateOrganizationMemberRole :one
UPDATE organization_members
SET role = $3
WHERE organization_id = $1 AND user_id = $2
RETURNING organization_id, user_id, role;