ALTER TABLE github_installations DROP COLUMN IF EXISTS state;
DROP TYPE IF EXISTS github_installation_state;