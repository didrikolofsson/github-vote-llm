DROP TRIGGER IF EXISTS organization_members_has_owner ON organization_members;
DROP FUNCTION IF EXISTS enforce_organization_has_owner();
DROP INDEX IF EXISTS organization_members_user_id_idx;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TYPE IF EXISTS organization_member_role;