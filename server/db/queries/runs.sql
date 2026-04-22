-- name: CreateRun :one
INSERT INTO feature_runs (
        prompt,
        feature_id,
        status,
        created_by_user_id,
        workspace
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
-- name: GetRunByID :one
SELECT fr.*,
    r.id AS repository_id,
    r.organization_id AS organization_id,
    r.name AS repository_name,
    r.owner AS repository_owner
FROM repositories AS r
    INNER JOIN features AS f ON f.repository_id = r.id
    INNER JOIN feature_runs AS fr ON fr.feature_id = f.id
WHERE fr.id = $1;
-- name: UpdateRunStatus :exec
UPDATE feature_runs
SET status = $1
WHERE id = $2;