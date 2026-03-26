-- ─── Auth ─────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    username   TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX users_email_idx ON users (email);

CREATE TABLE authorization_codes (
    id             BIGSERIAL PRIMARY KEY,
    code           TEXT NOT NULL UNIQUE,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_challenge TEXT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    used           BOOLEAN NOT NULL DEFAULT false,
    expires_at     TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE github_connections (
    user_id                BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE PRIMARY KEY,
    access_token_encrypted TEXT NOT NULL,
    refresh_token          TEXT,
    token_expires_at       TIMESTAMPTZ,
    github_user_id         BIGINT,
    github_login           TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─── Organizations ────────────────────────────────────────────────────────────

CREATE TABLE organizations (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT organization_name_unique UNIQUE (name)
);

CREATE TYPE organization_member_role AS ENUM ('owner', 'member');

CREATE TABLE organization_members (
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            organization_member_role NOT NULL DEFAULT 'member',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX organization_members_organization_id_idx ON organization_members (organization_id);
-- One organization per user
CREATE UNIQUE INDEX organization_members_user_id_idx ON organization_members (user_id);

CREATE OR REPLACE FUNCTION enforce_organization_has_owner()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM organizations WHERE id = OLD.organization_id) THEN
        RETURN OLD;
    END IF;
    IF TG_OP = 'DELETE' AND OLD.role = 'owner' THEN
        IF (SELECT COUNT(*) FROM organization_members
            WHERE organization_id = OLD.organization_id AND role = 'owner') <= 1 THEN
            RAISE EXCEPTION 'organization must have at least one owner';
        END IF;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.role = 'owner' AND NEW.role != 'owner' THEN
        IF (SELECT COUNT(*) FROM organization_members
            WHERE organization_id = OLD.organization_id AND role = 'owner') <= 1 THEN
            RAISE EXCEPTION 'organization must have at least one owner';
        END IF;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER organization_members_has_owner
BEFORE DELETE OR UPDATE OF role ON organization_members
FOR EACH ROW EXECUTE FUNCTION enforce_organization_has_owner();

-- ─── Repositories ─────────────────────────────────────────────────────────────

CREATE TABLE repositories (
    id              BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner           TEXT NOT NULL,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, owner, name)
);

CREATE INDEX repositories_organization_id_idx ON repositories (organization_id);

-- ─── Features ─────────────────────────────────────────────────────────────────

CREATE TYPE feature_status AS ENUM ('open', 'planned', 'in_progress', 'done', 'rejected');

CREATE TABLE features (
    id              BIGSERIAL PRIMARY KEY,
    repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          feature_status NOT NULL DEFAULT 'open',
    area            TEXT,
    roadmap_x       FLOAT,
    roadmap_y       FLOAT,
    roadmap_locked  BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX features_repository_id_idx ON features (repository_id);

CREATE TABLE feature_votes (
    id          BIGSERIAL PRIMARY KEY,
    feature_id  BIGINT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    voter_token TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (feature_id, voter_token)
);

CREATE TABLE feature_comments (
    id          BIGSERIAL PRIMARY KEY,
    feature_id  BIGINT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    author_name TEXT NOT NULL DEFAULT 'Anonymous',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE feature_dependencies (
    feature_id BIGINT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    depends_on BIGINT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    PRIMARY KEY (feature_id, depends_on)
);
