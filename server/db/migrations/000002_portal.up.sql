-- Add slug to organizations for public portal URLs
ALTER TABLE organizations ADD COLUMN slug TEXT UNIQUE;

-- Back-fill: strip non-alphanumeric (except spaces), lowercase, replace spaces with hyphens
UPDATE organizations
SET slug = lower(
    regexp_replace(
        regexp_replace(name, '[^a-zA-Z0-9\s]', '', 'g'),
        '\s+', '-', 'g'
    )
);

ALTER TABLE organizations ALTER COLUMN slug SET NOT NULL;

-- Add portal_public flag to repositories (off by default)
ALTER TABLE repositories ADD COLUMN portal_public BOOLEAN NOT NULL DEFAULT false;
