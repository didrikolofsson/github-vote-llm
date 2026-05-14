-- name: CreateRun :one
INSERT INTO
    feature_runs (
        prompt,
        feature_id,
        status,
        created_by_user_id,
        workspace
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    *;
-- name: GetRunByID :one
SELECT
    fr.*,
    r.id AS repository_id,
    r.organization_id AS organization_id,
    r.name AS repository_name,
    r.owner AS repository_owner
FROM
    organization_repositories AS r
    INNER JOIN features AS f ON f.repository_id = r.id
    INNER JOIN feature_runs AS fr ON fr.feature_id = f.id
WHERE
    fr.id = $1;

-- name: ListRunsByRepository :many
SELECT
    fr.id,
    fr.prompt,
    fr.feature_id,
    fr.status,
    fr.created_by_user_id,
    fr.created_at,
    fr.completed_at,
    NULL::TEXT AS pr_url
FROM
    feature_runs AS fr
    INNER JOIN features AS f ON f.id = fr.feature_id
WHERE
    f.repository_id = $1
ORDER BY
    fr.created_at DESC;

-- name: UpdateRunStatus :exec
UPDATE feature_runs SET status = $1, completed_at = CASE WHEN $1::feature_run_status IN ('completed', 'failed', 'cancelled') THEN now() ELSE completed_at END WHERE id = $2;

-- name: UpdateRunPRURL :exec
UPDATE feature_runs SET pr_url = $1 WHERE id = $2;

-- name: UpdateRunPID :exec
UPDATE feature_runs SET pid = $1 WHERE id = $2;

-- name: SetRunCancelled :exec
UPDATE feature_runs SET status = 'cancelled', completed_at = now() WHERE id = $1;

-- name: DeleteCancelledRun :exec
DELETE FROM feature_runs WHERE id = $1 AND status IN ('cancelled', 'failed');