-- name: GetRepoConfig :one
SELECT * FROM repo_config
WHERE owner = $1 AND repo = $2;

-- name: UpsertRepoConfig :one
INSERT INTO repo_config (
    owner, repo,
    label_approved, label_in_progress, label_done, label_failed, label_feature_request,
    vote_threshold, timeout_minutes, max_budget_usd, anthropic_api_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (owner, repo) DO UPDATE SET
    label_approved        = EXCLUDED.label_approved,
    label_in_progress     = EXCLUDED.label_in_progress,
    label_done            = EXCLUDED.label_done,
    label_failed          = EXCLUDED.label_failed,
    label_feature_request = EXCLUDED.label_feature_request,
    vote_threshold        = EXCLUDED.vote_threshold,
    timeout_minutes       = EXCLUDED.timeout_minutes,
    max_budget_usd        = EXCLUDED.max_budget_usd,
    anthropic_api_key     = EXCLUDED.anthropic_api_key,
    updated_at            = now()
RETURNING *;
-- name: ListRepoConfigs :many
SELECT * FROM repo_config ORDER BY created_at DESC;
