-- New enum types
CREATE TYPE review_status_type AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE build_status_type  AS ENUM ('pending', 'in_progress', 'stuck', 'done', 'rejected');
CREATE TYPE vote_urgency_type  AS ENUM ('blocking', 'important', 'nice_to_have');

-- Add new columns to features
ALTER TABLE features
  ADD COLUMN review_status review_status_type NOT NULL DEFAULT 'pending',
  ADD COLUMN build_status  build_status_type;

-- Migrate existing data: visible features become approved
UPDATE features SET review_status = 'approved'
  WHERE status IN ('open', 'planned', 'in_progress', 'done');

UPDATE features SET review_status = 'rejected'
  WHERE status = 'rejected';

-- Map old lifecycle status to build_status
UPDATE features SET build_status = 'pending'     WHERE status = 'open';
UPDATE features SET build_status = 'pending'     WHERE status = 'planned';
UPDATE features SET build_status = 'in_progress' WHERE status = 'in_progress';
UPDATE features SET build_status = 'done'        WHERE status = 'done';

-- Enforce: build_status can only be set when review_status is approved
ALTER TABLE features
  ADD CONSTRAINT build_status_requires_approval
  CHECK (build_status IS NULL OR review_status = 'approved');

-- Drop old status column and enum
ALTER TABLE features DROP COLUMN status;
DROP TYPE feature_status;

-- Add vote signal fields
ALTER TABLE feature_votes
  ADD COLUMN reason  TEXT NOT NULL DEFAULT '',
  ADD COLUMN urgency vote_urgency_type;
