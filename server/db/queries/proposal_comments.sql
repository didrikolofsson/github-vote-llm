-- name: ListProposalComments :many
SELECT * FROM proposal_comments WHERE proposal_id = $1 ORDER BY created_at ASC;

-- name: CreateProposalComment :one
INSERT INTO proposal_comments (proposal_id, body, author_name) VALUES ($1, $2, $3) RETURNING *;
