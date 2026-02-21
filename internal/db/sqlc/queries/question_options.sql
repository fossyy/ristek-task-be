-- name: CreateQuestionOption :one
INSERT INTO question_options (form_id, question_id, label, position)
VALUES ($1, $2, $3, $4)
    RETURNING *;

-- name: GetOptionsByFormID :many
SELECT * FROM question_options
WHERE form_id = $1
ORDER BY question_id, position ASC;

-- name: GetOptionsByQuestionID :many
SELECT * FROM question_options
WHERE form_id = $1 AND question_id = $2
ORDER BY position ASC;

-- name: DeleteOptionsByFormID :exec
DELETE FROM question_options
WHERE form_id = $1;