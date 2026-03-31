-- Restore original feature_status enum
CREATE TYPE feature_status AS ENUM ('open', 'planned', 'in_progress', 'done', 'rejected');

-- Re-add status column
ALTER TABLE features ADD COLUMN status feature_status NOT NULL DEFAULT 'open';

-- Restore data from new columns
UPDATE features SET status = 'rejected'    WHERE review_status = 'rejected';
UPDATE features SET status = 'open'        WHERE review_status = 'approved' AND build_status IS NULL;
UPDATE features SET status = 'open'        WHERE review_status = 'approved' AND build_status = 'pending';
UPDATE features SET status = 'in_progress' WHERE build_status = 'in_progress';
UPDATE features SET status = 'done'        WHERE build_status = 'done';

-- Remove constraint and new columns
ALTER TABLE features
  DROP CONSTRAINT build_status_requires_approval,
  DROP COLUMN review_status,
  DROP COLUMN build_status;

DROP TYPE review_status_type;
DROP TYPE build_status_type;

-- Remove vote signal fields
ALTER TABLE feature_votes
  DROP COLUMN reason,
  DROP COLUMN urgency;

DROP TYPE vote_urgency_type;
