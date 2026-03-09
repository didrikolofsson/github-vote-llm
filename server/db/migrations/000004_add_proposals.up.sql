CREATE TABLE proposals (
  id          BIGSERIAL PRIMARY KEY,
  owner       TEXT NOT NULL,
  repo        TEXT NOT NULL,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  vote_count  INT  NOT NULL DEFAULT 0,
  status      TEXT NOT NULL DEFAULT 'open',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX proposals_owner_repo_votes_idx ON proposals (owner, repo, vote_count DESC);

CREATE TABLE proposal_comments (
  id          BIGSERIAL PRIMARY KEY,
  proposal_id BIGINT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
  body        TEXT NOT NULL,
  author_name TEXT NOT NULL DEFAULT 'Anonymous',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE repo_config ADD COLUMN is_board_public BOOLEAN NOT NULL DEFAULT false;
