CREATE TYPE github_installation_state AS ENUM ('pending', 'active', 'suspended');
ALTER TABLE github_installations
ADD COLUMN state github_installation_state NOT NULL DEFAULT 'pending';