-- name: CreateExecution :one
INSERT INTO executions (owner, repo, issue_number, status)
VALUES ($1, $2, $3, 'pending')
RETURNING *;

-- name: UpdateExecutionInProgress :one
UPDATE executions
SET status = 'in_progress', branch = $1, updated_at = now()
WHERE id = $2
RETURNING *;

-- name: UpdateExecutionSuccess :one
UPDATE executions
SET status = 'success', pr_url = $1, updated_at = now()
WHERE id = $2
RETURNING *;

-- name: UpdateExecutionFailed :one
UPDATE executions
SET status = 'failed', error = $1, updated_at = now()
WHERE id = $2
RETURNING *;
