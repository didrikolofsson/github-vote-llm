CREATE TABLE organizations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TYPE organization_member_role AS ENUM ('owner', 'member');
CREATE TABLE organization_members (
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role organization_member_role NOT NULL DEFAULT 'member',
    PRIMARY KEY (organization_id, user_id)
);
-- Unique organization name
ALTER TABLE organizations
ADD CONSTRAINT organization_name_unique UNIQUE (name);
CREATE INDEX organization_members_organization_id_idx ON organization_members (organization_id);
-- One organization per user (can relax later for multi-org)
CREATE UNIQUE INDEX organization_members_user_id_idx ON organization_members (user_id);
-- Trigger to ensure at least one owner per organization
CREATE OR REPLACE FUNCTION enforce_organization_has_owner() RETURNS TRIGGER AS $$ BEGIN -- Skip check when the organization itself is being deleted (cascade)
    IF NOT EXISTS (SELECT 1 FROM organizations WHERE id = OLD.organization_id) THEN
        RETURN OLD;
    END IF;
-- Case 1: Deleting an owner
    IF TG_OP = 'DELETE'
    AND OLD.role = 'owner' THEN IF (
        SELECT COUNT(*)
        FROM organization_members
        WHERE organization_id = OLD.organization_id
            AND role = 'owner'
    ) <= 1 THEN RAISE EXCEPTION 'organization must have at least one owner';
END IF;
END IF;
-- Case 2: Downgrading an owner to member
IF TG_OP = 'UPDATE'
AND OLD.role = 'owner'
AND NEW.role != 'owner' THEN IF (
    SELECT COUNT(*)
    FROM organization_members
    WHERE organization_id = OLD.organization_id
        AND role = 'owner'
) <= 1 THEN RAISE EXCEPTION 'organization must have at least one owner';
END IF;
END IF;
RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER organization_members_has_owner BEFORE DELETE
OR
UPDATE OF role ON organization_members FOR EACH ROW EXECUTE FUNCTION enforce_organization_has_owner();