-- name: CreateFormResponse :one
INSERT INTO form_responses (form_id, user_id)
VALUES ($1, $2)
    RETURNING *;

-- name: GetFormResponsesByFormID :many
SELECT * FROM form_responses
WHERE form_id = $1
ORDER BY submitted_at DESC;

-- name: FormHasResponses :one
SELECT EXISTS (
    SELECT 1 FROM form_responses WHERE form_id = $1
) AS has_responses;

-- name: CreateResponseAnswer :one
INSERT INTO response_answers (response_id, question_id, form_id, answer_text)
VALUES ($1, $2, $3, $4)
    RETURNING *;

-- name: GetAnswersByResponseID :many
SELECT * FROM response_answers
WHERE response_id = $1
ORDER BY id ASC;
