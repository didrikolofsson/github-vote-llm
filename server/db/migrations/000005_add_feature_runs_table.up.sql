CREATE TYPE feature_run_status AS ENUM ('pending', 'running', 'completed', 'failed');

CREATE TABLE feature_runs (
    id BIGSERIAL PRIMARY KEY,
    prompt TEXT NOT NULL,
    feature_id BIGINT NOT NULL UNIQUE REFERENCES features(id) ON DELETE CASCADE,
    status feature_run_status NOT NULL DEFAULT 'pending',
    created_by_user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX feature_runs_feature_id_idx ON feature_runs (feature_id);
