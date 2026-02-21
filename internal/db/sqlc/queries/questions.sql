-- name: CreateQuestion :one
INSERT INTO questions (form_id, type, title, required, position)
VALUES ($1, $2, $3, $4, $5)
    RETURNING *;

-- name: GetQuestionsByFormID :many
SELECT * FROM questions
WHERE form_id = $1
ORDER BY position ASC;

-- name: DeleteQuestionsByFormID :exec
DELETE FROM questions
WHERE form_id = $1;